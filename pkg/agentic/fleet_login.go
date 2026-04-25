// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"

	core "dappco.re/go/core"
)

type fleetLoginOutput struct {
	Success     bool
	AgentID     string
	AgentAPIKey string
	ExpiresAt   string
	KeyPath     string
}

// result := subsystem.cmdFleetLogin(core.NewOptions(core.Option{Key: "_arg", Value: "123456"}))
func (s *PrepSubsystem) cmdFleetLogin(options core.Options) core.Result {
	code := core.Trim(optionStringValue(options, "code", "pairing_code", "pairing-code", "_arg"))
	if !fleetPairingCodeValid(code) {
		core.Print(nil, "usage: core-agent login <6-digit-code>")
		return core.Result{Value: core.E("agentic.cmdFleetLogin", "pairing code must be 6 digits", nil), OK: false}
	}

	result := s.loginWithPairingCode(s.commandContext(), core.NewOptions(
		core.Option{Key: "code", Value: code},
		core.Option{Key: "api", Value: optionStringValue(options, "api", "api_url", "api-url")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdFleetLogin", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(fleetLoginOutput)
	if !ok {
		err := core.E("agentic.cmdFleetLogin", "invalid fleet login output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "logged in")
	if output.AgentID != "" {
		core.Print(nil, "agent:      %s", output.AgentID)
	}
	if output.ExpiresAt != "" {
		core.Print(nil, "expires:    %s", output.ExpiresAt)
	}
	core.Print(nil, "saved to:   %s", output.KeyPath)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) loginWithPairingCode(ctx context.Context, options core.Options) core.Result {
	code := core.Trim(optionStringValue(options, "code", "pairing_code", "pairing-code", "_arg"))
	if !fleetPairingCodeValid(code) {
		return core.Result{Value: core.E("agentic.fleet.login", "pairing code must be 6 digits", nil), OK: false}
	}

	config := fleetClientConfig{
		APIURL: fleetAPIURLFromOptions(s, options),
	}

	result := s.fleetJSONRequest(ctx, "agentic.fleet.login", config, http.MethodPost, "/v1/device/pair", map[string]any{
		"code": code,
	})
	if !result.OK {
		return result
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		return core.Result{Value: core.E("agentic.fleet.login", "invalid fleet login payload", nil), OK: false}
	}

	output := parseFleetLoginOutput(payload)
	if output.AgentAPIKey == "" {
		return core.Result{Value: core.E("agentic.fleet.login", "device pairing response did not include an api key", nil), OK: false}
	}

	output.Success = true
	output.KeyPath = fleetAgentKeyPath()

	if ensureResult := fs.EnsureDir(core.PathDir(output.KeyPath)); !ensureResult.OK {
		err, _ := ensureResult.Value.(error)
		return core.Result{Value: core.E("agentic.fleet.login", "create fleet key directory", err), OK: false}
	}
	if writeResult := fs.WriteMode(output.KeyPath, output.AgentAPIKey, 0600); !writeResult.OK {
		err, _ := writeResult.Value.(error)
		return core.Result{Value: core.E("agentic.fleet.login", "write fleet api key", err), OK: false}
	}

	if s != nil {
		s.brainKey = output.AgentAPIKey
	}

	fleetRememberBase(fleetClientConfig{APIURL: config.APIURL, AgentID: output.AgentID, AgentAPIKey: output.AgentAPIKey})
	return core.Result{Value: output, OK: true}
}

func parseFleetLoginOutput(payload map[string]any) fleetLoginOutput {
	values := payloadDataMap(payload)
	if len(values) == 0 {
		values = payload
	}

	key := stringValue(values["agent_api_key"])
	if key == "" {
		key = stringValue(values["key"])
	}
	if key == "" {
		key = stringValue(anyMapValue(values["agent_api_key"])["key"])
	}

	return fleetLoginOutput{
		AgentAPIKey: key,
		AgentID:     stringValue(values["agent_id"]),
		ExpiresAt:   stringValue(values["expires_at"]),
	}
}

func fleetPairingCodeValid(code string) bool {
	if len(code) != 6 {
		return false
	}

	for _, character := range code {
		if character < '0' || character > '9' {
			return false
		}
	}

	return true
}
