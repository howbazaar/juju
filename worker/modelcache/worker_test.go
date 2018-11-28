// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelcache_test

import (
	"time"

	"github.com/juju/loggo"
	jc "github.com/juju/testing/checkers"
	"github.com/prometheus/client_golang/prometheus"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/worker.v1"
	"gopkg.in/juju/worker.v1/workertest"

	"github.com/juju/juju/core/cache"
	"github.com/juju/juju/core/life"
	"github.com/juju/juju/state"
	statetesting "github.com/juju/juju/state/testing"
	"github.com/juju/juju/testing"
	"github.com/juju/juju/worker/modelcache"
	workerstate "github.com/juju/juju/worker/state"
)

type WorkerSuite struct {
	statetesting.StateSuite
	logger loggo.Logger
	config workerstate.ManifoldConfig
	notify func(interface{})
}

var _ = gc.Suite(&WorkerSuite{})

func (s *WorkerSuite) SetUpTest(c *gc.C) {
	s.StateSuite.SetUpTest(c)
	s.logger = loggo.GetLogger("test")
	s.logger.SetLogLevel(loggo.TRACE)
}

func (s *WorkerSuite) start(c *gc.C) worker.Worker {
	config := modelcache.Config{
		Logger:               s.logger,
		StatePool:            s.StatePool,
		PrometheusRegisterer: noopRegisterer{},
		Cleanup:              func() {},
		Notify:               s.notify,
	}

	w, err := modelcache.NewWorker(config)
	c.Assert(err, jc.ErrorIsNil)
	s.AddCleanup(func(c *gc.C) {
		workertest.CleanKill(c, w)
	})
	return w
}

func (s *WorkerSuite) captureModelEvents(c *gc.C) <-chan interface{} {
	events := make(chan interface{})
	s.notify = func(change interface{}) {
		send := false
		switch change.(type) {
		case cache.ModelChange:
			send = true
		case cache.RemoveModel:
			send = true
		default:
			// no-op
		}
		if send {
			select {
			case events <- change:
			case <-time.After(testing.LongWait):
				c.Fatalf("change not processed by test")
			}
		}
	}
	return events
}

func (s *WorkerSuite) TestInitialModel(c *gc.C) {
	changes := s.captureModelEvents(c)
	s.start(c)

	var obtained interface{}
	select {
	case obtained = <-changes:
	case <-time.After(testing.LongWait):
		c.Fatalf("no change")
	}

	change, ok := obtained.(cache.ModelChange)
	c.Assert(ok, jc.IsTrue)
	expected, err := s.State.Model()
	c.Assert(err, jc.ErrorIsNil)
	c.Check(change.ModelUUID, gc.Equals, s.State.ModelUUID())
	c.Check(change.Name, gc.Equals, expected.Name())
	c.Check(change.Life, gc.Equals, life.Value(expected.Life().String()))
	c.Check(change.Owner, gc.Equals, expected.Owner().Name())
	cfg, err := expected.Config()
	c.Assert(err, jc.ErrorIsNil)
	c.Check(change.Config, jc.DeepEquals, cfg.AllAttrs())
	status, err := expected.Status()
	c.Assert(err, jc.ErrorIsNil)
	c.Check(change.Status, jc.DeepEquals, status)
}

func (s *WorkerSuite) TestNewModel(c *gc.C) {
	s.start(c)

	model := s.Factory.MakeModel(c, nil)
	s.State.StartSync()
	defer model.Close()

	<-time.After(time.Second)
	c.Fatalf("oops")
}

func (s *WorkerSuite) TestRemovedModel(c *gc.C) {
	s.start(c)

	st := s.Factory.MakeModel(c, nil)
	s.State.StartSync()
	defer st.Close()

	model, err := st.Model()
	c.Assert(err, jc.ErrorIsNil)

	c.Logf("\nDestroy\n\n")

	err = model.Destroy(state.DestroyModelParams{})
	c.Assert(err, jc.ErrorIsNil)
	s.State.StartSync()

	c.Logf("\nProcessDyingModel\n\n")

	err = st.ProcessDyingModel()
	c.Assert(err, jc.ErrorIsNil)
	s.State.StartSync()

	c.Logf("\nRemoveDyingModel\n\n")

	err = st.RemoveDyingModel()
	c.Assert(err, jc.ErrorIsNil)
	s.State.StartSync()

	<-time.After(time.Second)
	c.Fatalf("oops")
}

type noopRegisterer struct {
	prometheus.Registerer
}

func (noopRegisterer) Register(prometheus.Collector) error {
	return nil
}

func (noopRegisterer) Unregister(prometheus.Collector) bool {
	return true
}
