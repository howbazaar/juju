// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/pubsub"
	"github.com/juju/utils/clock"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/juju/worker.v1/catacomb"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/apiserver/apiserverhttp"
	"github.com/juju/juju/pubsub/apiserver"
	"github.com/juju/juju/pubsub/controller"
)

var logger = loggo.GetLogger("juju.worker.httpserver")

// Config is the configuration required for running an API server worker.
type Config struct {
	AgentConfig             agent.Config
	Clock                   clock.Clock
	TLSConfig               *tls.Config
	AutocertHandler         http.Handler
	AutocertListener        net.Listener
	Mux                     *apiserverhttp.Mux
	PrometheusRegisterer    prometheus.Registerer
	Hub                     *pubsub.StructuredHub
	ControllerAPIPort       int
	UpdateControllerAPIPort func(int) error
}

// Validate validates the API server configuration.
func (config Config) Validate() error {
	if config.AgentConfig == nil {
		return errors.NotValidf("nil AgentConfig")
	}
	if config.TLSConfig == nil {
		return errors.NotValidf("nil TLSConfig")
	}
	if config.Mux == nil {
		return errors.NotValidf("nil Mux")
	}
	if config.PrometheusRegisterer == nil {
		return errors.NotValidf("nil PrometheusRegisterer")
	}
	if config.AutocertHandler != nil && config.AutocertListener == nil {
		return errors.NewNotValid(nil, "AutocertListener must not be nil if AutocertHandler is not nil")
	}
	return nil
}

// NewWorker returns a new API server worker, with the given configuration.
func NewWorker(config Config) (*Worker, error) {
	if err := config.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	w := &Worker{
		config:              config,
		url:                 make(chan string),
		apiServerRegistered: make(chan struct{}),
	}

	servingInfo, ok := config.AgentConfig.StateServingInfo()
	if !ok {
		return nil, errors.New("missing state serving info")
	}
	w.apiPort = servingInfo.APIPort
	w.controllerAPIPort = servingInfo.ControllerAPIPort
	// We need to make sure that update the agent config with
	// the potentially new controller api port from the database
	// before we start any other workers that need to connect
	// to the controller over the controller port.
	w.updateControllerPort(w.config.ControllerAPIPort)

	// TODO (thumper): from 2.5 onwards we should also wait for the
	// raft transport worker to start. In 2.4 though it is possible that
	// it isn't running, so we ignore it here.

	// Declare the unsub function so the function closure in the Subscribe
	// call can use it.
	var err error
	var unsub func()
	unsub, err = config.Hub.Subscribe(apiserver.MuxRegisteredTopic, func(_ string, _ map[string]interface{}) {
		logger.Infof("told that apiserver is ready")
		close(w.apiServerRegistered)
		// Don't need to listen any more.
		unsub()
	})
	if err != nil {
		return nil, errors.Annotate(err, "unable to subscribe to apiserver registered topic.")
	}
	if err := catacomb.Invoke(catacomb.Plan{
		Site: &w.catacomb,
		Work: w.loop,
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return w, nil
}

type Worker struct {
	catacomb catacomb.Catacomb
	config   Config
	url      chan string

	// The http server and the api server have a mutual relationship.
	// This worker runs the http server, but all the end point handling
	// is done by the apiserver. If this worker starts the Serve method
	// before the registration is complete api clients can get a 404.
	apiServerRegistered chan struct{}

	apiPort           int
	controllerAPIPort int

	unsub func()
}

func (w *Worker) Kill() {
	w.catacomb.Kill(nil)
}

func (w *Worker) Wait() error {
	return w.catacomb.Wait()
}

// URL returns the base URL of the HTTP server of the form
// https://ipaddr:port with no trailing slash.
func (w *Worker) URL() string {
	select {
	case <-w.catacomb.Dying():
		return ""
	case url := <-w.url:
		return url
	}
}

func (w *Worker) updateControllerPort(port int) {
	if w.controllerAPIPort != port {
		// The local cache is out of date, update it.
		logger.Infof("updating controller API port to %v", port)
		err := w.config.UpdateControllerAPIPort(port)
		if err != nil {
			logger.Errorf("unable to update agent.conf with new controller API port: %v", err)
		}
		w.controllerAPIPort = port
	}
}

func (w *Worker) loop() error {

	unsub, err := w.config.Hub.Subscribe(controller.ConfigChanged,
		func(topic string, data controller.ConfigChangedMessage, err error) {
			w.updateControllerPort(data.Config.ControllerAPIPort())
		})
	if err != nil {
		return errors.Annotate(err, "unable to subscribe to details topic")
	}
	defer unsub()

	logger.Debugf("waiting for apiserver to finishing registering with mux")
	// Now wait for the apiserver to finish its registration of handler
	// endpoints in the mux.
	for {
		select {
		case <-w.catacomb.Dying():
			// The server isn't yet running.
			return w.catacomb.ErrDying()
		case w.url <- "": // Since we aren't yet listening, the url is blank.
		case <-w.apiServerRegistered:
			// Now open the ports and start the http server.
			break
		}
	}

	var listener listener
	if w.controllerAPIPort == 0 {
		listener, err = w.newSimpleListener()
	} else {
		listener, err = w.newDualPortListener()
	}
	if err != nil {
		return errors.Trace(err)
	}

	serverLog := log.New(&loggoWrapper{
		level:  loggo.WARNING,
		logger: logger,
	}, "", 0) // no prefix and no flags so log.Logger doesn't add extra prefixes
	server := &http.Server{
		Handler:   w.config.Mux,
		TLSConfig: w.config.TLSConfig,
		ErrorLog:  serverLog,
	}
	go server.Serve(tls.NewListener(listener, w.config.TLSConfig))
	defer func() {
		logger.Infof("shutting down HTTP server")
		// Shutting down the server will also close listener.
		err := server.Shutdown(context.Background())
		w.catacomb.Kill(err)
	}()

	if w.config.AutocertHandler != nil {
		autocertServer := &http.Server{
			Handler:  w.config.AutocertHandler,
			ErrorLog: serverLog,
		}
		go autocertServer.Serve(w.config.AutocertListener)
		defer func() {
			logger.Infof("shutting down autocert HTTP server")
			// This will also close the autocert listener.
			err := autocertServer.Shutdown(context.Background())
			w.catacomb.Kill(err)
		}()
	}

	for {
		select {
		case <-w.catacomb.Dying():
			// Asked to shutdown - make sure we wait until all clients
			// have finished up.
			w.config.Mux.Wait()
			return w.catacomb.ErrDying()
		case w.url <- listener.URL():
		}
	}
}

type listener interface {
	net.Listener
	URL() string
}

func (w *Worker) newSimpleListener() (listener, error) {
	listenAddr := net.JoinHostPort("", strconv.Itoa(w.apiPort))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, errors.Trace(err)
	}
	logger.Infof("listening on %q", listener.Addr())
	return &simpleListener{listener}, nil
}

type simpleListener struct {
	net.Listener
}

func (s *simpleListener) URL() string {
	return fmt.Sprintf("https://%s", s.Addr())
}

func (w *Worker) newDualPortListener() (listener, error) {
	readyDelay := time.Second
	if delay := w.config.AgentConfig.Value("CONTROLLER_READY_DELAY"); delay != "" {
		d, err := time.ParseDuration(delay)
		if err != nil {
			logger.Warningf("agent config file has bad value for CONTROLLER_READY_DELAY duration: %v", err)
		} else {
			readyDelay = d
		}
	}

	// Only open the controller port until we have been told that
	// the controller is ready. This is currently done by the event
	// from the peergrouper.
	// TODO (thumper): make the raft worker publish an event when
	// it knows who the raft master is. This means that this controller
	// is part of the concensus set, and when it is, is is OK to accept
	// agent connections. Until that time, accepting an agent connection
	// would be a bit of a waste of time.
	listenAddr := net.JoinHostPort("", strconv.Itoa(w.controllerAPIPort))
	listener, err := net.Listen("tcp", listenAddr)
	logger.Infof("listening for controller connections on %q", listener.Addr())
	dual := &dualListener{
		clock:              w.config.Clock,
		delay:              readyDelay,
		apiPort:            w.apiPort,
		controllerListener: listener,
		done:               make(chan struct{}),
		errors:             make(chan error),
		connections:        make(chan net.Conn),
	}
	go dual.accept(listener)

	dual.unsub, err = w.config.Hub.Subscribe(apiserver.DetailsTopic, dual.openAPIPort)
	if err != nil {
		dual.Close()
		return nil, errors.Annotate(err, "unable to subscribe to details topic")
	}

	return dual, err
}

type dualListener struct {
	clock   clock.Clock
	delay   time.Duration
	apiPort int

	controllerListener net.Listener
	apiListener        net.Listener

	mu     sync.Mutex
	closer sync.Once

	done        chan struct{}
	errors      chan error
	connections chan net.Conn

	unsub func()
}

func (d *dualListener) accept(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case d.errors <- err:
			case <-d.done:
				logger.Infof("no longer accepting connections on %s", listener.Addr())
				return
			}
		} else {
			select {
			case d.connections <- conn:
			case <-d.done:
				conn.Close()
				logger.Infof("no longer accepting connections on %s", listener.Addr())
				return
			}
		}
	}
}

// Accept implements net.Listener.
func (d *dualListener) Accept() (net.Conn, error) {
	select {
	case <-d.done:
		return nil, errors.New("listener has been closed")
	case err := <-d.errors:
		return nil, errors.Trace(err)
	case conn := <-d.connections:
		return conn, nil
	}
}

// Close implements net.Listener. Closes all the open listeners.
func (d *dualListener) Close() error {
	// Only close the channel once.
	d.closer.Do(func() { close(d.done) })
	err := d.controllerListener.Close()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.apiListener != nil {
		err2 := d.apiListener.Close()
		if err == nil {
			err = err2
		}
		// If we already have a close error, we don't really care
		// about this one.
	}
	return errors.Trace(err)
}

// Addr implements net.Listener. If the api port has been opened, we
// return that, otherwise we return the controller port address.
func (d *dualListener) Addr() net.Addr {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.apiListener != nil {
		return d.apiListener.Addr()
	}
	return d.controllerListener.Addr()
}

// URL implements the listener method.
func (d *dualListener) URL() string {
	return fmt.Sprintf("https://%s", d.Addr())
}

// openAPIPort opens the api port and starts accepting connections.
func (d *dualListener) openAPIPort(_ string, _ map[string]interface{}) {
	logger.Infof("waiting for %s before allowing api connections", d.delay)
	<-d.clock.After(d.delay)

	defer d.unsub()

	d.mu.Lock()
	defer d.mu.Unlock()
	// Make sure we haven't been closed already.
	select {
	case <-d.done:
		return
	default:
		// We are all good.
	}

	listenAddr := net.JoinHostPort("", strconv.Itoa(d.apiPort))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		select {
		case d.errors <- err:
		case <-d.done:
			logger.Errorf("can't open api port: %v, but worker exiting already", err)
		}
		return
	}

	logger.Infof("listening for api connections on %q", listener.Addr())
	go d.accept(listener)
}
