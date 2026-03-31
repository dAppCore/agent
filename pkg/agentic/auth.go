// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
)

// key := agentic.AgentAPIKey{ID: 7, Name: "codex local", Prefix: "ak_abcd", RateLimit: 60}
type AgentAPIKey struct {
	ID          int      `json:"id"`
	WorkspaceID int      `json:"workspace_id,omitempty"`
	Name        string   `json:"name,omitempty"`
	Key         string   `json:"key,omitempty"`
	Prefix      string   `json:"prefix,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	RateLimit   int      `json:"rate_limit,omitempty"`
	CallCount   int      `json:"call_count,omitempty"`
	LastUsedAt  string   `json:"last_used_at,omitempty"`
	ExpiresAt   string   `json:"expires_at,omitempty"`
	RevokedAt   string   `json:"revoked_at,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
}

// input := agentic.AuthProvisionInput{OAuthUserID: "user-42", Permissions: []string{"plans:read"}}
type AuthProvisionInput struct {
	OAuthUserID string   `json:"oauth_user_id"`
	Name        string   `json:"name,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	RateLimit   int      `json:"rate_limit,omitempty"`
	ExpiresAt   string   `json:"expires_at,omitempty"`
}

// out := agentic.AuthProvisionOutput{Success: true, Key: agentic.AgentAPIKey{Prefix: "ak_abcd"}}
type AuthProvisionOutput struct {
	Success bool        `json:"success"`
	Key     AgentAPIKey `json:"key"`
}

// input := agentic.AuthRevokeInput{KeyID: "7"}
type AuthRevokeInput struct {
	KeyID string `json:"key_id"`
}

// out := agentic.AuthRevokeOutput{Success: true, KeyID: "7", Revoked: true}
type AuthRevokeOutput struct {
	Success bool   `json:"success"`
	KeyID   string `json:"key_id"`
	Revoked bool   `json:"revoked"`
}

// result := c.Action("agentic.auth.provision").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "oauth_user_id", Value: "user-42"},
//	core.Option{Key: "permissions", Value: "plans:read,plans:write"},
//
// ))
func (s *PrepSubsystem) handleAuthProvision(ctx context.Context, options core.Options) core.Result {
	input := AuthProvisionInput{
		OAuthUserID: optionStringValue(options, "oauth_user_id", "oauth-user-id", "user_id", "user-id", "_arg"),
		Name:        optionStringValue(options, "name"),
		Permissions: optionStringSliceValue(options, "permissions"),
		RateLimit:   optionIntValue(options, "rate_limit", "rate-limit"),
		ExpiresAt:   optionStringValue(options, "expires_at", "expires-at"),
	}
	if input.OAuthUserID == "" {
		return core.Result{Value: core.E("agentic.auth.provision", "oauth_user_id is required", nil), OK: false}
	}

	body := map[string]any{
		"oauth_user_id": input.OAuthUserID,
	}
	if input.Name != "" {
		body["name"] = input.Name
	}
	if len(input.Permissions) > 0 {
		body["permissions"] = input.Permissions
	}
	if input.RateLimit > 0 {
		body["rate_limit"] = input.RateLimit
	}
	if input.ExpiresAt != "" {
		body["expires_at"] = input.ExpiresAt
	}

	result := s.platformPayload(ctx, "agentic.auth.provision", "POST", "/v1/agent/auth/provision", body)
	if !result.OK {
		return result
	}

	return core.Result{Value: AuthProvisionOutput{
		Success: true,
		Key:     parseAgentAPIKey(payloadResourceMap(result.Value.(map[string]any), "key", "api_key", "agent_api_key")),
	}, OK: true}
}

// result := c.Action("agentic.auth.revoke").Run(ctx, core.NewOptions(core.Option{Key: "key_id", Value: "7"}))
func (s *PrepSubsystem) handleAuthRevoke(ctx context.Context, options core.Options) core.Result {
	keyID := optionStringValue(options, "key_id", "key-id", "_arg")
	if keyID == "" {
		return core.Result{Value: core.E("agentic.auth.revoke", "key_id is required", nil), OK: false}
	}

	path := core.Concat("/v1/agent/auth/revoke/", keyID)
	result := s.platformPayload(ctx, "agentic.auth.revoke", "DELETE", path, nil)
	if !result.OK {
		return result
	}

	output := AuthRevokeOutput{
		Success: true,
		KeyID:   keyID,
		Revoked: true,
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		return core.Result{Value: output, OK: true}
	}

	if data := payloadResourceMap(payload, "result", "revocation"); len(data) > 0 {
		if value := stringValue(data["key_id"]); value != "" {
			output.KeyID = value
		}
		if value, ok := boolValueOK(data["revoked"]); ok {
			output.Revoked = value
		}
		if value, ok := boolValueOK(data["success"]); ok {
			output.Success = value
		}
		return core.Result{Value: output, OK: output.Success && output.Revoked}
	}

	if data, exists := payload["data"]; exists {
		if value, ok := boolValueOK(data); ok {
			output.Revoked = value
			return core.Result{Value: output, OK: output.Success && output.Revoked}
		}
	}

	return core.Result{Value: output, OK: true}
}

func parseAgentAPIKey(values map[string]any) AgentAPIKey {
	return AgentAPIKey{
		ID:          intValue(values["id"]),
		WorkspaceID: intValue(values["workspace_id"]),
		Name:        stringValue(values["name"]),
		Key:         stringValue(values["key"]),
		Prefix:      stringValue(values["prefix"]),
		Permissions: listValue(values["permissions"]),
		RateLimit:   intValue(values["rate_limit"]),
		CallCount:   intValue(values["call_count"]),
		LastUsedAt:  stringValue(values["last_used_at"]),
		ExpiresAt:   stringValue(values["expires_at"]),
		RevokedAt:   stringValue(values["revoked_at"]),
		CreatedAt:   stringValue(values["created_at"]),
	}
}

func boolValueOK(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		trimmed := core.Lower(core.Trim(typed))
		switch trimmed {
		case "true", "1", "yes":
			return true, true
		case "false", "0", "no":
			return false, true
		}
	case int:
		return typed != 0, true
	case int64:
		return typed != 0, true
	case float64:
		return typed != 0, true
	}
	return false, false
}
