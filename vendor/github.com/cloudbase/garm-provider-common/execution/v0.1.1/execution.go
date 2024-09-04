// Copyright 2023 Cloudbase Solutions SRL
//
//    Licensed under the Apache License, Version 2.0 (the "License"); you may
//    not use this file except in compliance with the License. You may obtain
//    a copy of the License at
//
//         http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//    WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//    License for the specific language governing permissions and limitations
//    under the License.

package executionv011

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/mod/semver"

	common "github.com/cloudbase/garm-provider-common/execution/common"
	"github.com/cloudbase/garm-provider-common/params"
)

func GetEnvironment() (EnvironmentV011, error) {
	env := EnvironmentV011{
		Command:            common.ExecutionCommand(os.Getenv("GARM_COMMAND")),
		ControllerID:       os.Getenv("GARM_CONTROLLER_ID"),
		PoolID:             os.Getenv("GARM_POOL_ID"),
		ProviderConfigFile: os.Getenv("GARM_PROVIDER_CONFIG_FILE"),
		InstanceID:         os.Getenv("GARM_INSTANCE_ID"),
		InterfaceVersion:   os.Getenv("GARM_INTERFACE_VERSION"),
		ExtraSpecs:         os.Getenv("GARM_POOL_EXTRASPECS"),
	}

	// If this is a CreateInstance command, we need to get the bootstrap params
	// from stdin
	boostrapParams, err := common.GetBoostrapParamsFromStdin(env.Command)
	if err != nil {
		return EnvironmentV011{}, fmt.Errorf("failed to get bootstrap params: %w", err)
	}
	env.BootstrapParams = boostrapParams

	if env.InterfaceVersion == "" {
		env.InterfaceVersion = common.Version010
	}

	if err := env.Validate(); err != nil {
		return EnvironmentV011{}, fmt.Errorf("failed to validate execution environment: %w", err)
	}

	return env, nil
}

type EnvironmentV011 struct {
	Command            common.ExecutionCommand
	ControllerID       string
	PoolID             string
	ProviderConfigFile string
	InstanceID         string
	InterfaceVersion   string
	ExtraSpecs         string
	BootstrapParams    params.BootstrapInstance
}

func (e EnvironmentV011) Validate() error {
	if e.Command == "" {
		return fmt.Errorf("missing GARM_COMMAND")
	}

	if e.ProviderConfigFile == "" {
		return fmt.Errorf("missing GARM_PROVIDER_CONFIG_FILE")
	}

	if _, err := os.Lstat(e.ProviderConfigFile); err != nil {
		return fmt.Errorf("error accessing config file: %w", err)
	}

	if e.ControllerID == "" {
		return fmt.Errorf("missing GARM_CONTROLLER_ID")
	}

	switch e.Command {
	case common.CreateInstanceCommand:
		if e.BootstrapParams.Name == "" {
			return fmt.Errorf("missing bootstrap params")
		}
		if e.ControllerID == "" {
			return fmt.Errorf("missing controller ID")
		}
		if e.PoolID == "" {
			return fmt.Errorf("missing pool ID")
		}
	case common.DeleteInstanceCommand, common.GetInstanceCommand,
		common.StartInstanceCommand, common.StopInstanceCommand:
		if e.InstanceID == "" {
			return fmt.Errorf("missing instance ID")
		}
		if e.PoolID == "" {
			return fmt.Errorf("missing pool ID")
		}
	case common.ListInstancesCommand:
		if e.PoolID == "" {
			return fmt.Errorf("missing pool ID")
		}
	case common.RemoveAllInstancesCommand:
		if e.ControllerID == "" {
			return fmt.Errorf("missing controller ID")
		}
	case common.GetVersionCommand:
		if !semver.IsValid(e.InterfaceVersion) {
			return fmt.Errorf("invalid interface version: %s", e.InterfaceVersion)
		}
	case common.ValidatePoolInfoCommand, common.GetConfigJSONSchemaCommand,
		common.GetExtraSpecsJSONSchemaCommand:
		if !semver.IsValid(e.InterfaceVersion) && e.InterfaceVersion == common.Version010 {
			return fmt.Errorf("invalid interface version: %s", e.InterfaceVersion)
		}
	default:
		return fmt.Errorf("unknown GARM_COMMAND: %s", e.Command)
	}
	return nil
}

func Run(ctx context.Context, provider ExternalProvider, env EnvironmentV011) (string, error) {
	var ret string
	switch env.Command {
	case common.CreateInstanceCommand:
		instance, err := provider.CreateInstance(ctx, env.BootstrapParams)
		if err != nil {
			return "", fmt.Errorf("failed to create instance in provider: %w", err)
		}

		asJs, err := json.Marshal(instance)
		if err != nil {
			return "", fmt.Errorf("failed to marshal response: %w", err)
		}
		ret = string(asJs)
	case common.GetInstanceCommand:
		instance, err := provider.GetInstance(ctx, env.InstanceID)
		if err != nil {
			return "", fmt.Errorf("failed to get instance from provider: %w", err)
		}
		asJs, err := json.Marshal(instance)
		if err != nil {
			return "", fmt.Errorf("failed to marshal response: %w", err)
		}
		ret = string(asJs)
	case common.ListInstancesCommand:
		instances, err := provider.ListInstances(ctx, env.PoolID)
		if err != nil {
			return "", fmt.Errorf("failed to list instances from provider: %w", err)
		}
		asJs, err := json.Marshal(instances)
		if err != nil {
			return "", fmt.Errorf("failed to marshal response: %w", err)
		}
		ret = string(asJs)
	case common.DeleteInstanceCommand:
		if err := provider.DeleteInstance(ctx, env.InstanceID); err != nil {
			return "", fmt.Errorf("failed to delete instance from provider: %w", err)
		}
	case common.RemoveAllInstancesCommand:
		if err := provider.RemoveAllInstances(ctx); err != nil {
			return "", fmt.Errorf("failed to destroy environment: %w", err)
		}
	case common.StartInstanceCommand:
		if err := provider.Start(ctx, env.InstanceID); err != nil {
			return "", fmt.Errorf("failed to start instance: %w", err)
		}
	case common.StopInstanceCommand:
		if err := provider.Stop(ctx, env.InstanceID, true); err != nil {
			return "", fmt.Errorf("failed to stop instance: %w", err)
		}
	case common.GetVersionCommand:
		version := provider.GetVersion(ctx)
		ret = string(version)
	case common.ValidatePoolInfoCommand:
		if err := provider.ValidatePoolInfo(ctx, env.BootstrapParams.Image, env.BootstrapParams.Flavor, env.ProviderConfigFile, env.ExtraSpecs); err != nil {
			return "", fmt.Errorf("failed to validate pool info: %w", err)
		}
	case common.GetConfigJSONSchemaCommand:
		schema, err := provider.GetConfigJSONSchema(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get config JSON schema: %w", err)
		}
		ret = schema
	case common.GetExtraSpecsJSONSchemaCommand:
		schema, err := provider.GetExtraSpecsJSONSchema(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get extra specs JSON schema: %w", err)
		}
		ret = schema
	default:
		return "", fmt.Errorf("invalid command: %s", env.Command)
	}
	return ret, nil
}
