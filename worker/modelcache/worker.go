// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelcache

import (
	"github.com/juju/errors"
	"github.com/kr/pretty"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/juju/worker.v1"
	"gopkg.in/juju/worker.v1/catacomb"

	"github.com/juju/juju/core/cache"
	"github.com/juju/juju/core/life"
	"github.com/juju/juju/core/status"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/multiwatcher"
)

// Config describes the necessary fields for NewWorker.
type Config struct {
	Logger               Logger
	StatePool            *state.StatePool
	PrometheusRegisterer prometheus.Registerer
	Cleanup              func()
	// Notify is used primarily for testing, and is passed through
	// to the cache.Controller. It is called every time the controller
	// processes an event.
	Notify func(interface{})
}

// Validate ensures all the necessary values are specified
func (c *Config) Validate() error {
	if c.Logger == nil {
		return errors.NotValidf("missing logger")
	}
	if c.StatePool == nil {
		return errors.NotValidf("missing state pool")
	}
	if c.PrometheusRegisterer == nil {
		return errors.NotValidf("missing prometheus registerer")
	}
	if c.Cleanup == nil {
		return errors.NotValidf("missing cleanup func")
	}
	return nil
}

type cacheWorker struct {
	config     Config
	catacomb   catacomb.Catacomb
	controller *cache.Controller
	changes    chan interface{}
}

// NewWorker creates a new cacheWorker, and starts an
// all model watcher.
func NewWorker(config Config) (worker.Worker, error) {
	if err := config.Validate(); err != nil {
		return nil, errors.Trace(err)
	}
	w := &cacheWorker{
		config:  config,
		changes: make(chan interface{}),
	}
	w.controller = cache.NewController(w.changes, config.Notify)
	if err := catacomb.Invoke(catacomb.Plan{
		Site: &w.catacomb,
		Work: w.loop,
		Init: []worker.Worker{w.controller},
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return w, nil
}

// Report returns information that is used in the dependency engine report.
func (c *cacheWorker) Report() map[string]interface{} {
	return c.controller.Report()
}

func (c *cacheWorker) loop() error {
	defer c.config.Cleanup()
	pool := c.config.StatePool
	allWatcher := pool.SystemState().WatchAllModels(pool)
	defer allWatcher.Stop()

	collector := cache.NewMetricsCollector(c.controller)
	c.config.PrometheusRegisterer.Register(collector)
	defer c.config.PrometheusRegisterer.Unregister(collector)

	watcherChanges := make(chan []multiwatcher.Delta)
	go func() {
		for {
			deltas, err := allWatcher.Next()
			if err != nil {
				if errors.Cause(err) == state.ErrStopped {
					return
				} else {
					c.catacomb.Kill(err)
					return
				}
			}
			select {
			case <-c.catacomb.Dying():
				return
			case watcherChanges <- deltas:
			}
		}
	}()

	for {
		select {
		case <-c.catacomb.Dying():
			return c.catacomb.ErrDying()
		case deltas := <-watcherChanges:
			// Process changes and send info down changes channel
			for _, d := range deltas {
				c.config.Logger.Tracef(pretty.Sprint(d))
				value := c.translate(d)
				if value != nil {
					select {
					case c.changes <- value:
					case <-c.catacomb.Dying():
						return nil
					}
				}
			}
		}
	}
}

func coreStatus(info multiwatcher.StatusInfo) status.StatusInfo {
	return status.StatusInfo{
		Status:  info.Current,
		Message: info.Message,
		Data:    info.Data,
		Since:   info.Since,
	}
}

func (c *cacheWorker) translate(d multiwatcher.Delta) interface{} {
	id := d.Entity.EntityId()
	switch id.Kind {
	case "model":
		if d.Removed {
			return cache.RemoveModel{
				ModelUUID: id.ModelUUID,
			}
		}
		value, ok := d.Entity.(*multiwatcher.ModelInfo)
		if !ok {
			c.config.Logger.Errorf("unexpected type %T", d.Entity)
			return nil
		}
		return cache.ModelChange{
			ModelUUID: value.ModelUUID,
			Name:      value.Name,
			Life:      life.Value(value.Life),
			Owner:     value.Owner,
			Config:    value.Config,
			Status:    coreStatus(value.Status),
			// TODO: constraints, sla
		}

	default:
		return nil
	}
}

// Kill is part of the worker.Worker interface.
func (c *cacheWorker) Kill() {
	c.catacomb.Kill(nil)
}

// Wait is part of the worker.Worker interface.
func (c *cacheWorker) Wait() error {
	return c.catacomb.Wait()
}
