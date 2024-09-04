// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright 2023 Cloudbase Solutions SRL
//
// Licensed under the AGPLv3, see LICENCE file for details

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	commonExecution "github.com/cloudbase/garm-provider-common/execution/common"
	execution "github.com/cloudbase/garm-provider-common/execution/v0.1.0"

	"github.com/cloudbase/garm-provider-lxd/provider"
)

var signals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
}

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), signals...)
	defer stop()

	executionEnv, err := execution.GetEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	prov, err := provider.NewLXDProvider(executionEnv.ProviderConfigFile, executionEnv.ControllerID, executionEnv.InterfaceVersion)
	if err != nil {
		log.Fatal(err)
	}

	result, err := execution.Run(ctx, prov, executionEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to run command: %s", err)
		os.Exit(commonExecution.ResolveErrorToExitCode(err))
	}
	if len(result) > 0 {
		fmt.Fprint(os.Stdout, result)
	}
}
