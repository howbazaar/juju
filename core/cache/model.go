// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cache

import (
	"sync"

	"gopkg.in/tomb.v2"

	"github.com/juju/juju/core/watcher"
	"github.com/juju/pubsub"
)

const modelConfigChange = "model-config-change"

// Model is a cached model in the controller. The model is kept up to
// date with changes flowing into the cached controller.
type Model struct {
	hub *pubsub.SimpleHub
	mu  sync.Mutex

	details    ModelChange
	configHash string
}

// modelTopic prefixes the topic with the model UUID.
func (m *Model) modelTopic(topic string) string {
	return m.details.ModelUUID + ":" + topic
}

func (m *Model) setDetails(details ModelChange) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.details = details

	configHash, err := hash(details.Config)
	if err != nil {
		logger.Errorf("invariant error - model config should be yaml serializable and hashable, %v", err)
		return
	}
	if configHash != m.configHash {
		m.configHash = configHash
		m.hub.Publish(m.modelTopic(modelConfigChange), details.Config)
	}
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
func (m *Model) WatchConfig(keys ...string) watcher.NotifyWatcher {
	// We use a single entry buffered channel for the changes.
	// This allows the config changed handler to send a value when there
	// is a change, but if that value hasn't been consumed before the
	// next change, the second change is discarded.
	watcher := &modelConfigWatcher{
		keys:    keys,
		changes: make(chan struct{}, 1),
	}
	watcher.hash = watcher.generateHash(m.Config())
	// Send initial event down the channel. We know that this will
	// execute immediately because it is a buffered channel.
	watcher.changes <- struct{}{}

	unsub := m.hub.Subscribe(m.modelTopic(modelConfigChange), watcher.configChanged)

	watcher.tomb.Go(func() error {
		<-watcher.tomb.Dying()
		unsub()
		return nil
	})

	return watcher
}

type modelConfigWatcher struct {
	keys    []string
	hash    string
	tomb    tomb.Tomb
	changes chan struct{}
}

func (w *modelConfigWatcher) generateHash(config map[string]interface{}) string {
	if len(w.keys) == 0 {
		return ""
	}
	interested := make(map[string]interface{})
	for _, key := range w.keys {
		interested[key] = config[key]
	}
	h, err := hash(interested)
	if err != nil {
		logger.Errorf("invariant error - model config should be yaml serializable and hashable, %v", err)
		return ""
	}
	return h
}

func (w *modelConfigWatcher) configChanged(topic string, value interface{}) {
	config, ok := value.(map[string]interface{})
	if !ok {
		logger.Errorf("programming error, value not a map[string]interface{}")
	}
	// TODO: consider a cached hashing for keys so we only generate
	// the hash once for any particular set of keys. Would mean we send
	// an object through that has the config and the hash sets.
	hash := w.generateHash(config)
	if hash != "" && hash == w.hash {
		// Nothing that we care about has changed, so we're done.
		return
	}
	// Let the listener know.
	select {
	case w.changes <- struct{}{}:
	default:
		// Already a pending change, so do nothing.
	}
}

// Changes is part of the core watcher definition.
// The changes channel is never closed.
func (w *modelConfigWatcher) Changes() watcher.NotifyChannel {
	return w.changes
}

// Kill is part of the worker.Worker interface.
func (w *modelConfigWatcher) Kill() {
	w.tomb.Kill(nil)
}

// Wait is part of the worker.Worker interface.
func (w *modelConfigWatcher) Wait() error {
	return w.tomb.Wait()
}
