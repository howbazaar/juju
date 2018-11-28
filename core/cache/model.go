// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cache

import (
	"sync"

	"gopkg.in/tomb.v2"

	"github.com/juju/juju/core/watcher"
)

// Model is a cached model in the controller. The model is kept up to
// date with changes flowing into the cached controller.
type Model struct {
	details ModelChange
	mu      sync.Mutex
}

// Config returns the current model config.
func (m *Model) Config() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.details.Config
}

// WatchConfig creates a watcher for the model config.
// If keys are specified, the watcher is only signals a change when
// those keys change values.
// XXX: consider strings watcher based on xtian's latest work with hashes.
func (m *Model) WatchConfig(keys ...string) watcher.NotifyWatcher {

	return &modelConfigWatcher{keys: keys}
}

type modelConfigWatcher struct {
	keys []string
	tomb tomb.Tomb
}

// XXX: implementation is currently illegal as nil isn't valid.
func (w *modelConfigWatcher) Changes() watcher.NotifyChannel {
	return nil
}

// Kill is part of the worker.Worker interface.
func (w *modelConfigWatcher) Kill() {
	w.tomb.Kill(nil)
}

// Wait is part of the worker.Worker interface.
func (w *modelConfigWatcher) Wait() error {
	return w.tomb.Wait()
}
