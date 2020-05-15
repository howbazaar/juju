// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver

import (
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/pubsub"

	"github.com/juju/juju/pubsub/apiserver"
)

// APIConnections represents a collection of the current API connections
// to the API Server.
type APIConnections struct {
	connections map[uint64]connection
	modelUUID   string
	hub         *pubsub.StructuredHub
	mux         pubsub.Multiplexer
	mu          sync.Mutex
	logger      loggo.Logger
}

type connection struct {
	conn       io.Closer
	modelUUID  string
	tag        string
	controller bool
}

// NewAPIConnections creates an APIConnections instance that is fully
// initialized. The uuid represents the controller model uuid. This is
// to acurately represent connections from the controller itself, or the
// controller model agents.
func NewAPIConnections(uuid string, hub *pubsub.StructuredHub) (*APIConnections, error) {
	mux, err := hub.NewMultiplexer()
	if err != nil {
		return nil, errors.Trace(err)
	}
	conn := &APIConnections{
		connections: make(map[uint64]connection),
		modelUUID:   uuid,
		hub:         hub,
		mux:         mux,
		logger:      loggo.GetLogger("juju.apiserver.connections"),
	}
	mux.Add(apiserver.ConnectTopic, conn.onConnect)
	return conn, nil
}

// Report is used to populate the engine report of the API Server
// with information about the api connections.
func (c *APIConnections) Report() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	other := 0
	controller := 0
	agent := 0
	for _, conn := range c.connections {
		switch {
		case conn.modelUUID == c.modelUUID:
			controller++
		case conn.controller:
			controller++
		case conn.tag != "":
			agent++
		default:
			// Other includes user connections, or connections
			// from agents that are currently established but not
			// yet logged in.
			other++
		}
	}
	report := map[string]interface{}{
		"other":      other,
		"controller": controller,
		"agent":      agent,
	}
	return report
}

// Close will unsubscribe the pubsub subscribers.
func (c *APIConnections) Close() {
	c.mux.Unsubscribe()
}

// Add is called with a unique connID for any particular server
// when the api handler is invoked for a given websocket connection.
func (c *APIConnections) Add(connID uint64, conn io.Closer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Tracef("add %d", connID)
	c.connections[connID] = connection{conn: conn}
}

// Remove is called when the websocket connection has been closed.
func (c *APIConnections) Remove(connID uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Tracef("remove %d", connID)
	delete(c.connections, connID)
}

func (c *APIConnections) onConnect(topic string, data apiserver.APIConnection, err error) {
	if err != nil {
		c.logger.Errorf("onConnect error %v", err)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if conn, found := c.connections[data.ConnectionID]; found {
		c.logger.Tracef("update %d, tag %q, controller %v", data.ConnectionID, data.AgentTag, data.ControllerAgent)
		conn.tag = data.AgentTag
		conn.modelUUID = data.ModelUUID
		conn.controller = data.ControllerAgent
		c.connections[data.ConnectionID] = conn
	}
	// It is possible, due to the async nature of pubsub, that the connection
	// may have been added, and then removed, before we get to process the
	// connection event that was published. This isn't really a problem, so
	// we don't do anything in that situation.
}
