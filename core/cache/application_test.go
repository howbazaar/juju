// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.
package cache_test

import (
	"github.com/juju/loggo"
	"github.com/juju/pubsub"
	"github.com/juju/testing"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/core/cache"
)

type ApplicationSuite struct {
	testing.IsolationSuite

	gauges *cache.ControllerGauges

	hub *pubsub.SimpleHub
}

var _ = gc.Suite(&ApplicationSuite{})

func (s *ApplicationSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)
	s.gauges = cache.CreateControllerGauges()
	logger := loggo.GetLogger("test")
	logger.SetLogLevel(loggo.TRACE)
	s.hub = pubsub.NewSimpleHub(&pubsub.SimpleHubConfig{
		Logger: logger,
	})
}
