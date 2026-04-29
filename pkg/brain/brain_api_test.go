// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"testing"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/ide"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func listedToolNames(t *testing.T, register func(*coremcp.Service)) []string {
	t.Helper()

	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)
	register(svc)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := svc.Server().Connect(context.Background(), serverTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	result, err := clientSession.ListTools(context.Background(), nil)
	core.RequireNoError(t, err)

	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func TestBridge_New_Good(t *testing.T) {
	bridge := ide.NewBridge(nil, ide.Config{})
	sub := New(bridge)
	core.AssertNotNil(t, sub)
	core.AssertSame(t, bridge, sub.bridge)
}

func TestNilBridge_New_Bad(t *testing.T) {
	sub := New(nil)
	core.AssertNotNil(t, sub)
	core.AssertNil(t, sub.bridge)
}

func TestFreshInstance_New_Ugly(t *testing.T) {
	first := New(nil)
	second := New(nil)
	core.AssertTrue(t, first != second)
	core.AssertEqual(t, "brain", first.Name())
}

func TestCustomEnv_NewDirect_Good(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "https://custom.api.test")
	t.Setenv("CORE_BRAIN_KEY", "test-key-123")
	t.Setenv("CORE_HOME", t.TempDir())

	sub := NewDirect()
	core.AssertEqual(t, "https://custom.api.test", sub.apiURL)
	core.AssertEqual(t, "test-key-123", sub.apiKey)
}

func TestMissingKey_NewDirect_Bad(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CORE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	sub := NewDirect()
	core.AssertEqual(t, "https://api.lthn.sh", sub.apiURL)
	core.AssertEmpty(t, sub.apiKey)
}

func TestHomeFallback_NewDirect_Ugly(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CORE_HOME", "")
	t.Setenv("DIR_HOME", "")

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	keyDir := core.JoinPath(tmpHome, ".claude")
	core.RequireTrue(t, fs.EnsureDir(keyDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(keyDir, "brain.key"), "  home-key-789  \n").OK)

	sub := NewDirect()
	core.AssertEqual(t, "home-key-789", sub.apiKey)
	core.AssertNotNil(t, sub.apiClient)
}

func TestNilCore_Register_Bad(t *testing.T) {
	result := Register(nil)
	core.AssertTrue(t, result.OK)
	sub, ok := result.Value.(*DirectSubsystem)
	core.RequireTrue(t, ok)
	core.AssertNil(t, sub.Core())
}

func TestFreshInstance_Register_Ugly(t *testing.T) {
	c := core.New()
	first := Register(c)
	second := Register(c)
	core.RequireTrue(t, first.OK)
	core.RequireTrue(t, second.OK)
	core.AssertTrue(t, first.Value != second.Value)
}

func TestDefaultSubsystem_Subsystem_Name_Good(t *testing.T) {
	got := New(nil).Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotEmpty(t, got)
}

func TestZeroValueSubsystem_Subsystem_Name_Bad(t *testing.T) {
	got := (&Subsystem{}).Name()
	core.AssertEqual(t, "brain", got)
	core.AssertContains(t, got, "brain")
}

func TestNilSubsystem_Subsystem_Name_Ugly(t *testing.T) {
	var sub *Subsystem
	got := sub.Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotContains(t, got, "/")
}

func TestDefaultSubsystem_Subsystem_RegisterTools_Good(t *testing.T) {
	names := listedToolNames(t, New(nil).RegisterTools)
	core.AssertContains(t, names, "brain_remember")
	core.AssertContains(t, names, "brain_list")
}

func TestZeroValueSubsystem_Subsystem_RegisterTools_Bad(t *testing.T) {
	names := listedToolNames(t, (&Subsystem{}).RegisterTools)
	core.AssertContains(t, names, "brain_recall")
	core.AssertContains(t, names, "brain_forget")
}

func TestRepeatedSubsystem_Subsystem_RegisterTools_Ugly(t *testing.T) {
	names := listedToolNames(t, func(svc *coremcp.Service) {
		sub := New(nil)
		sub.RegisterTools(svc)
		sub.RegisterTools(svc)
	})
	core.AssertContains(t, names, "brain_remember")
	core.AssertContains(t, names, "brain_recall")
}

func TestDefaultSubsystem_Subsystem_Shutdown_Good(t *testing.T) {
	err := New(nil).Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestZeroValueSubsystem_Subsystem_Shutdown_Bad(t *testing.T) {
	err := (&Subsystem{}).Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestNilSubsystem_Subsystem_Shutdown_Ugly(t *testing.T) {
	var sub *Subsystem
	core.AssertNotPanics(t, func() {
		core.AssertNoError(t, sub.Shutdown(context.Background()))
	})
}

func TestConstructedDirect_DirectSubsystem_Name_Good(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "https://custom.api.test")
	t.Setenv("CORE_BRAIN_KEY", "test-key")
	got := NewDirect().Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotEmpty(t, got)
}

func TestZeroValueDirect_DirectSubsystem_Name_Bad(t *testing.T) {
	got := (&DirectSubsystem{}).Name()
	core.AssertEqual(t, "brain", got)
	core.AssertContains(t, got, "brain")
}

func TestNilDirect_DirectSubsystem_Name_Ugly(t *testing.T) {
	var sub *DirectSubsystem
	got := sub.Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotContains(t, got, "/")
}

func TestRuntimeActions_DirectSubsystem_OnStartup_Good(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "https://api.lthn.sh")
	t.Setenv("CORE_BRAIN_KEY", "test-key")
	c := core.New()
	sub := NewDirect()
	sub.ServiceRuntime = core.NewServiceRuntime(c, DirectOptions{})

	result := sub.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, c.Action("brain.remember").Exists())
}

func TestNilRuntime_DirectSubsystem_OnStartup_Bad(t *testing.T) {
	result := (&DirectSubsystem{}).OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestNilCoreRuntime_DirectSubsystem_OnStartup_Ugly(t *testing.T) {
	sub := &DirectSubsystem{ServiceRuntime: core.NewServiceRuntime(nil, DirectOptions{})}
	result := sub.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestDirectTools_DirectSubsystem_RegisterMessagingTools_Good(t *testing.T) {
	names := listedToolNames(t, NewDirect().RegisterMessagingTools)
	core.AssertContains(t, names, "agent_send")
	core.AssertContains(t, names, "agent_conversation")
}

func TestZeroValueDirect_DirectSubsystem_RegisterMessagingTools_Bad(t *testing.T) {
	names := listedToolNames(t, (&DirectSubsystem{}).RegisterMessagingTools)
	core.AssertContains(t, names, "agent_send")
	core.AssertContains(t, names, "agent_inbox")
}

func TestRepeatedDirect_DirectSubsystem_RegisterMessagingTools_Ugly(t *testing.T) {
	names := listedToolNames(t, func(svc *coremcp.Service) {
		sub := &DirectSubsystem{}
		sub.RegisterMessagingTools(svc)
		sub.RegisterMessagingTools(svc)
	})
	core.AssertContains(t, names, "agent_inbox")
	core.AssertContains(t, names, "agent_conversation")
}

func TestDirectTools_DirectSubsystem_RegisterTools_Good(t *testing.T) {
	names := listedToolNames(t, NewDirect().RegisterTools)
	core.AssertContains(t, names, "brain_remember")
	core.AssertContains(t, names, "agent_send")
}

func TestZeroValueDirect_DirectSubsystem_RegisterTools_Bad(t *testing.T) {
	names := listedToolNames(t, (&DirectSubsystem{}).RegisterTools)
	core.AssertContains(t, names, "brain_list")
	core.AssertContains(t, names, "agent_inbox")
}

func TestRepeatedDirect_DirectSubsystem_RegisterTools_Ugly(t *testing.T) {
	names := listedToolNames(t, func(svc *coremcp.Service) {
		sub := &DirectSubsystem{}
		sub.RegisterTools(svc)
		sub.RegisterTools(svc)
	})
	core.AssertContains(t, names, "brain_recall")
	core.AssertContains(t, names, "agent_conversation")
}

func TestDirectShutdown_DirectSubsystem_Shutdown_Good(t *testing.T) {
	err := NewDirect().Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestZeroValueDirect_DirectSubsystem_Shutdown_Bad(t *testing.T) {
	err := (&DirectSubsystem{}).Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestNilDirect_DirectSubsystem_Shutdown_Ugly(t *testing.T) {
	var sub *DirectSubsystem
	core.AssertNotPanics(t, func() {
		core.AssertNoError(t, sub.Shutdown(context.Background()))
	})
}
