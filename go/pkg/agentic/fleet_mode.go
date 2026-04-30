// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"

	core "dappco.re/go"
)

func (s *PrepSubsystem) registerFleetCommands() {
	c := s.Core()
	c.Command("login", core.Command{Description: "Exchange a 6-digit pairing code for a fleet api key", Action: s.cmdFleetLogin})
	c.Command("agentic:login", core.Command{Description: "Exchange a 6-digit pairing code for a fleet api key", Action: s.cmdFleetLogin})
	c.Command("fleet", core.Command{Description: "Run or inspect fleet mode", Action: s.cmdFleet})
	c.Command("agentic:fleet", core.Command{Description: "Run or inspect fleet mode", Action: s.cmdFleet})
	c.Command("fleet/nodes", core.Command{Description: "List registered fleet nodes", Action: s.cmdFleetNodesCommand})
	c.Command("agentic:fleet/nodes", core.Command{Description: "List registered fleet nodes", Action: s.cmdFleetNodesCommand})
	c.Command("fleet/status", core.Command{Description: "Show current fleet connection status", Action: s.cmdFleetStatus})
	c.Command("agentic:fleet/status", core.Command{Description: "Show current fleet connection status", Action: s.cmdFleetStatus})
}

func (s *PrepSubsystem) cmdFleet(options core.Options) core.Result {
	action := optionStringValue(options, "action", "_arg")
	switch action {
	case "nodes":
		return s.cmdFleetNodesCommand(options)
	case "status":
		return s.cmdFleetStatus(options)
	case "", "help":
		if optionBoolValue(options, "help") || optionStringValue(options, "agent_id", "agent-id") == "" {
			printFleetUsage()
			return core.Result{OK: true}
		}

		config := fleetClientConfigFromOptions(s, options)
		core.Print(nil, "fleet mode")
		core.Print(nil, "api:        %s", config.APIURL)
		core.Print(nil, "agent:      %s", config.AgentID)
		result := s.Connect(s.commandContext(), options)
		if !result.OK {
			err := commandResultError("agentic.cmdFleet", result)
			core.Print(nil, "error: %v", err)
			return core.Result{Value: err, OK: false}
		}
		return result
	default:
		printFleetUsage()
		return core.Result{Value: core.E("agentic.cmdFleet", core.Concat("unknown fleet command: ", action), nil), OK: false}
	}
}

func (s *PrepSubsystem) cmdFleetNodesCommand(options core.Options) core.Result {
	result := s.listFleetNodes(s.commandContext(), options)
	if !result.OK {
		err := commandResultError("agentic.cmdFleetNodes", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(FleetNodesOutput)
	if !ok {
		err := core.E("agentic.cmdFleetNodes", "invalid fleet nodes output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if len(output.Nodes) == 0 {
		core.Print(nil, "no fleet nodes")
		return core.Result{Value: output, OK: true}
	}

	for _, node := range output.Nodes {
		core.Print(nil, "  %-10s %-8s %-10s %s", node.Status, node.Platform, node.AgentID, core.Join(",", node.Models...))
	}
	core.Print(nil, "")
	core.Print(nil, "total: %d", output.Total)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdFleetStatus(options core.Options) core.Result {
	snapshot := fleetRuntimeSnapshotValue()
	config := fleetClientConfigFromOptions(s, options)

	if snapshot.APIURL == "" {
		snapshot.APIURL = config.APIURL
	}
	if snapshot.AgentID == "" {
		snapshot.AgentID = config.AgentID
	}
	if snapshot.State == "" {
		snapshot.State = "offline"
	}
	if snapshot.Transport == "" {
		snapshot.Transport = "none"
	}

	core.Print(nil, "api:             %s", snapshot.APIURL)
	core.Print(nil, "agent:           %s", snapshot.AgentID)
	core.Print(nil, "state:           %s", snapshot.State)
	core.Print(nil, "transport:       %s", snapshot.Transport)
	if snapshot.LastConnectedAt != "" {
		core.Print(nil, "last connected:  %s", snapshot.LastConnectedAt)
	}
	if snapshot.LastHeartbeatAt != "" {
		core.Print(nil, "last heartbeat:  %s", snapshot.LastHeartbeatAt)
	} else {
		core.Print(nil, "last heartbeat:  never")
	}
	if snapshot.LastEventAt != "" {
		core.Print(nil, "last event:      %s", snapshot.LastEventAt)
	}
	if fleetTaskSummary(snapshot.LastTask) != "" {
		core.Print(nil, "last task:       %s", fleetTaskSummary(snapshot.LastTask))
		if snapshot.LastTaskReceived != "" {
			core.Print(nil, "task received:   %s", snapshot.LastTaskReceived)
		}
	} else {
		core.Print(nil, "last task:       none")
	}
	if snapshot.LastError != "" {
		core.Print(nil, "last error:      %s", snapshot.LastError)
	}

	return core.Result{Value: snapshot, OK: true}
}

func (s *PrepSubsystem) listFleetNodes(ctx context.Context, options core.Options) core.Result {
	config := fleetClientConfigFromOptions(s, options)

	path := "/v1/fleet/nodes"
	path = appendQueryParam(path, "status", optionStringValue(options, "status"))
	path = appendQueryParam(path, "platform", optionStringValue(options, "platform"))

	result := s.fleetJSONRequest(ctx, "agentic.fleet.nodes", config, http.MethodGet, path, nil)
	if !result.OK {
		return result
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		return core.Result{Value: core.E("agentic.fleet.nodes", "invalid fleet nodes payload", nil), OK: false}
	}

	return core.Result{Value: parseFleetNodesOutput(payload), OK: true}
}

func printFleetUsage() {
	core.Print(nil, "usage: core-agent fleet --api=https://api.lthn.ai --agent-id=charon")
	core.Print(nil, "       core-agent fleet nodes [--status=online] [--platform=linux]")
	core.Print(nil, "       core-agent fleet status")
}

func fleetTaskSummary(task FleetTask) string {
	if task.ID == 0 && task.Repo == "" && task.Task == "" {
		return ""
	}

	summary := ""
	if task.ID > 0 {
		summary = core.Sprintf("#%d", task.ID)
	}
	if task.Repo != "" {
		if summary != "" {
			summary = core.Concat(summary, " ")
		}
		summary = core.Concat(summary, task.Repo)
	}
	if task.Status != "" {
		if summary != "" {
			summary = core.Concat(summary, " ")
		}
		summary = core.Concat(summary, task.Status)
	}
	if task.Task != "" {
		if summary != "" {
			summary = core.Concat(summary, " ")
		}
		summary = core.Concat(summary, task.Task)
	}
	return summary
}
