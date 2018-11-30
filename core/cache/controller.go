// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cache

import (
	"sync"

	"github.com/juju/errors"
	"gopkg.in/tomb.v2"
)

// Controller is the primary cached object.
type Controller struct {
	changes <-chan interface{}
	tomb    tomb.Tomb
	mu      sync.Mutex
	models  map[string]*Model
	// notify func is used primarily for testing, and is called
	// every time a change has been processed, and the change is
	// passed as the arg.
	notify func(interface{})
}

// NewController creates a new cached controller intance.
// The changes channel is what is used to supply the cache with the changes
// in order for the cache to be kept up to date.
func NewController(changes <-chan interface{}, notify func(interface{})) *Controller {
	c := &Controller{
		changes: changes,
		models:  make(map[string]*Model),
		notify:  notify,
	}
	c.tomb.Go(c.loop)
	return c
}

func (c *Controller) loop() error {
	for {
		select {
		case <-c.tomb.Dying():
			return nil
		case change := <-c.changes:
			switch ch := change.(type) {
			case ModelChange:
				c.UpdateModel(ch)
			case RemoveModel:
				c.RemoveModel(ch)
			}
			if c.notify != nil {
				c.notify(change)
			}
		}
	}
}

// ModelUUIDs returns the UUIDs of the models in the cache.
func (c *Controller) ModelUUIDs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]string, 0, len(c.models))
	for uuid := range c.models {
		result = append(result, uuid)
	}
	return result
}

// Kill is part of the worker.Worker interface.
func (c *Controller) Kill() {
	c.tomb.Kill(nil)
}

// Wait is part of the worker.Worker interface.
func (c *Controller) Wait() error {
	return c.tomb.Wait()
}

// Model return the model for the specified UUID.
// If the model isn't found, a NotFoundError is returned.
// A copy of the model is returned.
func (c *Controller) Model(uuid string) (*Model, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	model, found := c.models[uuid]
	if !found {
		return nil, errors.NotFoundf("model %q", uuid)
	}
	result := &Model{
		details: model.details,
	}
	return result, nil
}

// UpdateModel will add or update the model details as
// described in the ModelChange.
func (c *Controller) UpdateModel(ch ModelChange) {
	c.mu.Lock()
	defer c.mu.Unlock()

	model, found := c.models[ch.ModelUUID]
	if !found {
		c.models[ch.ModelUUID] = &Model{
			details: ch,
		}
	} else {
		model.mu.Lock()
		model.details = ch
		model.mu.Unlock()
	}
}

// RemoveModel removes the model from the cache.
func (c *Controller) RemoveModel(ch RemoveModel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.models, ch.ModelUUID)
}
