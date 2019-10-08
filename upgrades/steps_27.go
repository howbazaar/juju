// Copyright 2019 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgrades

import (
	"github.com/juju/juju/service"
)

// stateStepsFor27 returns upgrade steps for Juju 2.7.0.
func stateStepsFor27() []Step {
	return []Step{
		&upgradeStep{
			description: "add controller node docs",
			targets:     []Target{DatabaseMaster},
			run: func(context Context) error {
				return context.State().AddControllerNodeDocs()
			},
		},
		&upgradeStep{
			description: "recreate spaces with IDs",
			targets:     []Target{DatabaseMaster},
			run: func(context Context) error {
				return context.State().AddSpaceIdToSpaceDocs()
			},
		},
		&upgradeStep{
			description: "change subnet AvailabilityZone to AvailabilityZones",
			targets:     []Target{DatabaseMaster},
			run: func(context Context) error {
				return context.State().ChangeSubnetAZtoSlice()
			},
		},
		&upgradeStep{
			description: "change subnet SpaceName to SpaceID",
			targets:     []Target{DatabaseMaster},
			run: func(context Context) error {
				return context.State().ChangeSubnetSpaceNameToSpaceID()
			},
		},
		&upgradeStep{
			description: "recreate subnets with IDs",
			targets:     []Target{DatabaseMaster},
			run: func(context Context) error {
				return context.State().AddSubnetIdToSubnetDocs()
			},
		},
		&upgradeStep{
			description: "replace portsDoc.SubnetID as a CIDR with an ID.",
			targets:     []Target{DatabaseMaster},
			run: func(context Context) error {
				return context.State().ReplacePortsDocSubnetIDCIDR()
			},
		},
		&upgradeStep{
			description: "ensure application settings exist for all relations",
			targets:     []Target{DatabaseMaster},
			run: func(context Context) error {
				return context.State().EnsureRelationApplicationSettings()
			},
		}, &upgradeStep{
			description: "change owner of unit and machine logs to adm",
			targets:     []Target{AllMachines},
			run: func(context Context) error {
				return context.State().AddModelLogfileMaxSize()
			},
		}, &upgradeStep{
			description: "change owner of unit and machine logs to adm",
			targets:     []Target{AllMachines},
			run:         resetLogPermissions,
		},
	}
}

// This adds upgrade steps, we just rewrite the default values which are set before.
// With this we can make sure that things are changed in one default place
func resetLogPermissions(context Context) error {
	sysdManager := service.NewServiceManagerWithDefaults()
	err := sysdManager.WriteServiceFiles()
	if err != nil {
		logger.Errorf("unsuccessful writing the service files in /lib/systemd/system path")
		return err
	}
	return nil
}
