// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package featuretests

import (
	"time"

	"github.com/juju/juju/core/cache/cachetest"
	"github.com/juju/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/api/logger"
	"github.com/juju/juju/core/watcher/watchertest"
	jujutesting "github.com/juju/juju/juju/testing"
	"github.com/juju/juju/state"
)

type apiLoggerSuite struct {
	jujutesting.JujuConnSuite
}

func (s *apiLoggerSuite) TestLoggingConfig(c *gc.C) {
	root, machine := s.OpenAPIAsNewMachine(c, state.JobHostUnits)
	logging := logger.NewState(root)

	obtained, err := logging.LoggingConfig(machine.Tag())
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(obtained, gc.Equals, "<root>=DEBUG;unit=DEBUG")
}

func (s *apiLoggerSuite) TestWatchLoggingConfig(c *gc.C) {
	root, machine := s.OpenAPIAsNewMachine(c, state.JobHostUnits)
	logging := logger.NewState(root)

	watcher, err := logging.WatchLoggingConfig(machine.Tag())
	c.Assert(err, jc.ErrorIsNil)
	_ = watcher

	wc := watchertest.NewNotifyWatcherC(c, watcher, nil)
	// Initial event.
	wc.AssertOneChange()

	change := cachetest.ModelChangeFromState(c, s.State)
	change.Config["logging-config"] = "juju=INFO;test=TRACE"
	select {
	case s.ControllerChangesChannel <- change:
	case <-time.After(testing.LongWait):
		c.Fatalf("unable to sent change to controller")
	}

	wc.AssertOneChange()
	wc.AssertStops()
}
