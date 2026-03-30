// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"strconv"

	"dappco.re/go/core/api"
	"dappco.re/go/core/api/pkg/provider"
	"forge.lthn.ai/core/go-ws"
	"forge.lthn.ai/core/mcp/pkg/mcp/ide"
	"github.com/gin-gonic/gin"
)

// provider := brain.NewProvider(bridge, hub)
// core.Println(provider.BasePath()) // "/api/brain"
type BrainProvider struct {
	bridge *ide.Bridge
	hub    *ws.Hub
}

var (
	_ provider.Provider    = (*BrainProvider)(nil)
	_ provider.Streamable  = (*BrainProvider)(nil)
	_ provider.Describable = (*BrainProvider)(nil)
	_ provider.Renderable  = (*BrainProvider)(nil)
)

const (
	statusOK                  = 200
	statusBadRequest          = 400
	statusInternalServerError = 500
	statusServiceUnavailable  = 503
)

// p := brain.NewProvider(bridge, hub)
// core.Println(p.BasePath())
func NewProvider(bridge *ide.Bridge, hub *ws.Hub) *BrainProvider {
	return &BrainProvider{
		bridge: bridge,
		hub:    hub,
	}
}

// name := p.Name() // "brain"
func (p *BrainProvider) Name() string { return "brain" }

// base := p.BasePath() // "/api/brain"
func (p *BrainProvider) BasePath() string { return "/api/brain" }

// channels := p.Channels()
// core.Println(channels[0]) // "brain.remember.complete"
func (p *BrainProvider) Channels() []string {
	return []string{
		"brain.remember.complete",
		"brain.recall.complete",
		"brain.forget.complete",
	}
}

// spec := p.Element()
// core.Println(spec.Tag) // "core-brain-panel"
func (p *BrainProvider) Element() provider.ElementSpec {
	return provider.ElementSpec{
		Tag:    "core-brain-panel",
		Source: "/assets/brain-panel.js",
	}
}

// p.RegisterRoutes(router.Group("/api/brain"))
func (p *BrainProvider) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/remember", p.remember)
	rg.POST("/recall", p.recall)
	rg.POST("/forget", p.forget)
	rg.GET("/list", p.list)
	rg.GET("/status", p.status)
}

// routes := p.Describe()
// core.Println(routes[0].Path) // "/remember"
func (p *BrainProvider) Describe() []api.RouteDescription {
	return []api.RouteDescription{
		{
			Method:      "POST",
			Path:        "/remember",
			Summary:     "Store a memory",
			Description: "Store a memory in the shared OpenBrain knowledge store via the Laravel backend.",
			Tags:        []string{"brain"},
			RequestBody: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content":    map[string]any{"type": "string"},
					"type":       map[string]any{"type": "string"},
					"tags":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"project":    map[string]any{"type": "string"},
					"confidence": map[string]any{"type": "number"},
				},
				"required": []string{"content", "type"},
			},
			Response: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"success":   map[string]any{"type": "boolean"},
					"memoryId":  map[string]any{"type": "string"},
					"timestamp": map[string]any{"type": "string", "format": "date-time"},
				},
			},
		},
		{
			Method:      "POST",
			Path:        "/recall",
			Summary:     "Semantic search memories",
			Description: "Semantic search across the shared OpenBrain knowledge store.",
			Tags:        []string{"brain"},
			RequestBody: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"top_k": map[string]any{"type": "integer"},
					"filter": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"project": map[string]any{"type": "string"},
							"type":    map[string]any{"type": "string"},
						},
					},
				},
				"required": []string{"query"},
			},
			Response: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"success":  map[string]any{"type": "boolean"},
					"count":    map[string]any{"type": "integer"},
					"memories": map[string]any{"type": "array"},
				},
			},
		},
		{
			Method:      "POST",
			Path:        "/forget",
			Summary:     "Remove a memory",
			Description: "Permanently delete a memory from the knowledge store.",
			Tags:        []string{"brain"},
			RequestBody: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string"},
					"reason": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Response: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"success":   map[string]any{"type": "boolean"},
					"forgotten": map[string]any{"type": "string"},
				},
			},
		},
		{
			Method:      "GET",
			Path:        "/list",
			Summary:     "List memories",
			Description: "List memories with optional filtering by project, type, and agent.",
			Tags:        []string{"brain"},
			Response: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"success":  map[string]any{"type": "boolean"},
					"count":    map[string]any{"type": "integer"},
					"memories": map[string]any{"type": "array"},
				},
			},
		},
		{
			Method:      "GET",
			Path:        "/status",
			Summary:     "Brain bridge status",
			Description: "Returns whether the Laravel bridge is connected.",
			Tags:        []string{"brain"},
			Response: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"connected": map[string]any{"type": "boolean"},
				},
			},
		},
	}
}

func (p *BrainProvider) remember(c *gin.Context) {
	if p.bridge == nil {
		p.respondBridgeUnavailable(c)
		return
	}

	var input RememberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		p.respondInvalidInput(c, err)
		return
	}

	err := p.bridge.Send(ide.BridgeMessage{
		Type: "brain_remember",
		Data: map[string]any{
			"content":    input.Content,
			"type":       input.Type,
			"tags":       input.Tags,
			"project":    input.Project,
			"confidence": input.Confidence,
			"supersedes": input.Supersedes,
			"expires_in": input.ExpiresIn,
		},
	})
	if err != nil {
		p.respondBridgeError(c, err)
		return
	}

	p.emitEvent("brain.remember.complete", map[string]any{
		"type":    input.Type,
		"project": input.Project,
	})

	c.JSON(statusOK, api.OK(map[string]any{"success": true}))
}

func (p *BrainProvider) recall(c *gin.Context) {
	if p.bridge == nil {
		p.respondBridgeUnavailable(c)
		return
	}

	var input RecallInput
	if err := c.ShouldBindJSON(&input); err != nil {
		p.respondInvalidInput(c, err)
		return
	}

	err := p.bridge.Send(ide.BridgeMessage{
		Type: "brain_recall",
		Data: map[string]any{
			"query":  input.Query,
			"top_k":  input.TopK,
			"filter": input.Filter,
		},
	})
	if err != nil {
		p.respondBridgeError(c, err)
		return
	}

	p.emitEvent("brain.recall.complete", map[string]any{
		"query": input.Query,
	})

	c.JSON(statusOK, api.OK(RecallOutput{
		Success:  true,
		Memories: []Memory{},
	}))
}

func (p *BrainProvider) forget(c *gin.Context) {
	if p.bridge == nil {
		p.respondBridgeUnavailable(c)
		return
	}

	var input ForgetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		p.respondInvalidInput(c, err)
		return
	}

	err := p.bridge.Send(ide.BridgeMessage{
		Type: "brain_forget",
		Data: map[string]any{
			"id":     input.ID,
			"reason": input.Reason,
		},
	})
	if err != nil {
		p.respondBridgeError(c, err)
		return
	}

	p.emitEvent("brain.forget.complete", map[string]any{
		"id": input.ID,
	})

	c.JSON(statusOK, api.OK(map[string]any{
		"success":   true,
		"forgotten": input.ID,
	}))
}

func (p *BrainProvider) list(c *gin.Context) {
	if p.bridge == nil {
		p.respondBridgeUnavailable(c)
		return
	}

	limit := 0
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			c.JSON(statusBadRequest, api.Fail("invalid_limit", "limit must be an integer"))
			return
		}
		limit = parsedLimit
	}

	err := p.bridge.Send(ide.BridgeMessage{
		Type: "brain_list",
		Data: map[string]any{
			"project":  c.Query("project"),
			"type":     c.Query("type"),
			"agent_id": c.Query("agent_id"),
			"limit":    limit,
		},
	})
	if err != nil {
		p.respondBridgeError(c, err)
		return
	}

	c.JSON(statusOK, api.OK(ListOutput{
		Success:  true,
		Memories: []Memory{},
	}))
}

func (p *BrainProvider) status(c *gin.Context) {
	connected := false
	if p.bridge != nil {
		connected = p.bridge.Connected()
	}
	c.JSON(statusOK, api.OK(map[string]any{
		"connected": connected,
	}))
}

func (p *BrainProvider) respondBridgeUnavailable(c *gin.Context) {
	c.JSON(statusServiceUnavailable, api.Fail("bridge_unavailable", "brain bridge not available"))
}

func (p *BrainProvider) respondInvalidInput(c *gin.Context, err error) {
	c.JSON(statusBadRequest, api.Fail("invalid_input", err.Error()))
}

func (p *BrainProvider) respondBridgeError(c *gin.Context, err error) {
	c.JSON(statusInternalServerError, api.Fail("bridge_error", err.Error()))
}

func (p *BrainProvider) emitEvent(channel string, data any) {
	if p.hub == nil {
		return
	}
	_ = p.hub.SendToChannel(channel, ws.Message{
		Type: ws.TypeEvent,
		Data: data,
	})
}
