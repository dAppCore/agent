// SPDX-License-Identifier: EUPL-1.2

package brain

import "github.com/gin-gonic/gin"

type Provider interface {
	Name() string
	BasePath() string
	RegisterRoutes(*gin.RouterGroup)
}

type Streamable interface {
	Provider
	Channels() []string
}

type Describable interface {
	Provider
	Describe() []RouteDescription
}

type Renderable interface {
	Element() ElementSpec
}

type ElementSpec struct {
	Tag    string `json:"tag"`
	Source string `json:"source"`
}

type RouteDescription struct {
	Method      string   `json:"method"`
	Path        string   `json:"route"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	RequestBody any      `json:"request_body,omitempty"`
	Response    any      `json:"response,omitempty"`
}

type providerResponse[T any] struct {
	Success bool           `json:"success"`
	Data    T              `json:"data,omitempty"`
	Error   *providerError `json:"error,omitempty"`
}

type providerError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func ok[T any](data T) providerResponse[T] {
	return providerResponse[T]{
		Success: true,
		Data:    data,
	}
}

func fail(code, message string) providerResponse[any] {
	return providerResponse[any]{
		Success: false,
		Error: &providerError{
			Code:    code,
			Message: message,
		},
	}
}
