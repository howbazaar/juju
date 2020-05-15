// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver_test

import (
	"time"

	"github.com/juju/loggo"
	"github.com/juju/pubsub"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver"
	ps "github.com/juju/juju/pubsub/apiserver"
)

var _ = gc.Suite(&APIConnectionsSuite{})

type APIConnectionsSuite struct {
	testing.IsolationSuite

	hub  *pubsub.StructuredHub
	conn *apiserver.APIConnections
}

func (s *APIConnectionsSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)
	s.hub = pubsub.NewStructuredHub(&pubsub.StructuredHubConfig{
		Logger: loggo.GetLogger("test.hub"),
	})
	conn, err := apiserver.NewAPIConnections("controller-model-uuid", s.hub)
	c.Assert(err, jc.ErrorIsNil)
	s.conn = conn
}

func (s *APIConnectionsSuite) checkReport(c *gc.C, agent, controller, other int) {
	report := s.conn.Report()
	c.Check(report, jc.DeepEquals, map[string]interface{}{
		"agent":      agent,
		"controller": controller,
		"other":      other,
	})
}

func (s *APIConnectionsSuite) TestInitial(c *gc.C) {
	s.checkReport(c, 0, 0, 0)
}

func (s *APIConnectionsSuite) TestAddOne(c *gc.C) {
	s.conn.Add(1, &closer{})
	s.checkReport(c, 0, 0, 1)
}

func (s *APIConnectionsSuite) TestLoginToControllerModel(c *gc.C) {
	s.conn.Add(1, &closer{})
	handled, err := s.hub.Publish(ps.ConnectTopic, ps.APIConnection{
		ConnectionID: 1,
		ModelUUID:    "controller-model-uuid",
		AgentTag:     "controller-0",
	})
	c.Assert(err, jc.ErrorIsNil)
	s.waitUntilHandled(c, handled)
	s.checkReport(c, 0, 1, 0)
}

func (s *APIConnectionsSuite) TestLoginByControllerToOtherModel(c *gc.C) {
	s.conn.Add(1, &closer{})
	handled, err := s.hub.Publish(ps.ConnectTopic, ps.APIConnection{
		ConnectionID:    1,
		ModelUUID:       "other-model-uuid",
		ControllerAgent: true,
		AgentTag:        "controller-0",
	})
	c.Assert(err, jc.ErrorIsNil)
	s.waitUntilHandled(c, handled)
	s.checkReport(c, 0, 1, 0)
}

func (s *APIConnectionsSuite) TestLoginToOtherModel(c *gc.C) {
	s.conn.Add(1, &closer{})
	handled, err := s.hub.Publish(ps.ConnectTopic, ps.APIConnection{
		ConnectionID: 1,
		ModelUUID:    "other-model-uuid",
		AgentTag:     "controller-0",
	})
	c.Assert(err, jc.ErrorIsNil)
	s.waitUntilHandled(c, handled)
	s.checkReport(c, 1, 0, 0)
}

func (s *APIConnectionsSuite) waitUntilHandled(c *gc.C, handled <-chan struct{}) {
	select {
	case <-handled:
	case <-time.After(testing.LongWait):
		c.Fatalf("event not handled")
	}
}

type closer struct {
	closed bool
}

func (c *closer) Close() error {
	c.closed = true
	return nil
}
