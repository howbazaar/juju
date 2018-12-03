// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cachetest

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/core/cache"
	"github.com/juju/juju/core/life"
	"github.com/juju/juju/state"
)

// ModelChangeFromState returns a ModelChange representing the current
// model for the state object.
func ModelChangeFromState(c *gc.C, st *state.State) cache.ModelChange {
	model, err := st.Model()
	c.Assert(err, jc.ErrorIsNil)
	return ModelChange(c, model)
}

// ModelChange returns a ModelChange representing the current state of the model.
func ModelChange(c *gc.C, model *state.Model) cache.ModelChange {
	change := cache.ModelChange{
		ModelUUID: model.UUID(),
		Name:      model.Name(),
		Life:      life.Value(model.Life().String()),
		Owner:     model.Owner().Name(),
	}
	config, err := model.Config()
	c.Assert(err, jc.ErrorIsNil)
	change.Config = config.AllAttrs()
	status, err := model.Status()
	c.Assert(err, jc.ErrorIsNil)
	change.Status = status
	return change
}
