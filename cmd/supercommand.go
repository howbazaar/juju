// Copyright 2012 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/juju/cmd"
	"github.com/juju/loggo"
	"github.com/juju/os/series"
	"github.com/juju/utils/arch"
	"github.com/juju/version"

	"github.com/juju/juju/juju/osenv"
	jujuversion "github.com/juju/juju/version"
)

func init() {
	// If the environment key is empty, ConfigureLoggers returns nil and does
	// nothing.
	err := loggo.ConfigureLoggers(os.Getenv(osenv.JujuStartupLoggingConfigEnvKey))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR parsing %s: %s\n\n", osenv.JujuStartupLoggingConfigEnvKey, err)
	}
}

var logger = loggo.GetLogger("juju.cmd")

// NewSuperCommand is like cmd.NewSuperCommand but
// it adds juju-specific functionality:
// - The default logging configuration is taken from the environment;
// - The version is configured to the current juju version;
// - The command emits a log message when a command runs.
func NewSuperCommand(p cmd.SuperCommandParams) *cmd.SuperCommand {
	p.Log = &cmd.Log{
		DefaultConfig: os.Getenv(osenv.JujuLoggingConfigEnvKey),
	}
	current := version.Binary{
		Number: jujuversion.Current,
		Arch:   arch.HostArch(),
		Series: series.MustHostSeries(),
	}

	// p.Version should be a version.Binary, but juju/cmd does not
	// import juju/juju/version so this cannot happen. We have
	// tests to assert that this string value is correct.
	p.Version = current.String()
	p.NotifyRun = runNotifier
	p.FlagKnownAs = "option"
	return cmd.NewSuperCommand(p)
}

func runNotifier(name string) {
	logger.Infof("running %s [%s %s %s]", name, jujuversion.Current, runtime.Compiler, runtime.Version())
	logger.Debugf("  args: %#v", os.Args)
}

func Info(i *cmd.Info) *cmd.Info {
	info := *i
	info.FlagKnownAs = "option"
	return &info
}
