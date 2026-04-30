// SPDX-License-Identifier: EUPL-1.2

package agentcompat

import (
	"context"
	"net/http"

	core "dappco.re/go"
	"gopkg.in/yaml.v3"
)

type ConcurrencyLimit struct {
	Total  int
	Models map[string]int
}

func (c *ConcurrencyLimit) UnmarshalYAMLResult(value *yaml.Node) core.Result {
	var n int
	if err := value.Decode(&n); err == nil {
		c.Total = n
		return core.Ok(nil)
	}
	var m map[string]int
	if err := value.Decode(&m); err != nil {
		return core.Fail(err)
	}
	c.Total = m["total"]
	c.Models = make(map[string]int)
	for key, entry := range m {
		if key != "total" {
			c.Models[key] = entry
		}
	}
	return core.Ok(nil)
}

func (c *ConcurrencyLimit) UnmarshalYAML(value *yaml.Node) error { // yaml contract
	result := c.UnmarshalYAMLResult(value)
	if result.OK {
		return nil
	}
	if err, ok := result.Value.(error); ok {
		return err
	}
	return core.E("ConcurrencyLimit.UnmarshalYAML", "decode failed", nil)
}

type HTTPStream struct {
	Client   *http.Client
	URL      string
	Token    string
	Method   string
	Response []byte
}

func (s *HTTPStream) SendResult(data []byte) core.Result {
	if s == nil {
		return core.Fail(core.E("HTTPStream.Send", "stream is required", nil))
	}
	if s.Client == nil {
		return core.Fail(core.E("HTTPStream.Send", "client is required", nil))
	}
	requestResult := core.NewHTTPRequestContext(context.Background(), s.Method, s.URL, core.NewReader(string(data)))
	if !requestResult.OK {
		return requestResult
	}
	request := requestResult.Value.(*core.Request)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if s.Token != "" {
		request.Header.Set("Authorization", core.Concat("token ", s.Token))
	}

	response, err := s.Client.Do(request)
	if err != nil {
		return core.Fail(err)
	}

	readResult := core.ReadAll(response.Body)
	if !readResult.OK {
		err, _ := readResult.Value.(error)
		return core.Fail(core.E("httpStream.Send", "failed to read response", err))
	}
	s.Response = []byte(readResult.Value.(string))
	return core.Ok(nil)
}

func (s *HTTPStream) Send(data []byte) error { // core.Stream contract
	if s == nil || s.Client == nil {
		panic(core.E("HTTPStream.Send", "stream client is required", nil))
	}
	result := s.SendResult(data)
	if result.OK {
		return nil
	}
	if err, ok := result.Value.(error); ok {
		return err
	}
	return core.E("HTTPStream.Send", "send failed", nil)
}

func (s *HTTPStream) ReceiveResult() core.Result {
	if s == nil {
		return core.Fail(core.E("HTTPStream.Receive", "stream is required", nil))
	}
	return core.Ok(s.Response)
}

func (s *HTTPStream) Receive() ([]byte, error) { // core.Stream contract
	if s == nil {
		panic(core.E("HTTPStream.Receive", "stream is required", nil))
	}
	result := s.ReceiveResult()
	if result.OK {
		return result.Value.([]byte), nil
	}
	if err, ok := result.Value.(error); ok {
		return nil, err
	}
	return nil, core.E("HTTPStream.Receive", "receive failed", nil)
}

func (s *HTTPStream) CloseResult() core.Result {
	return core.Ok(nil)
}

func (s *HTTPStream) Close() error { // core.Stream contract
	result := s.CloseResult()
	if result.OK {
		return nil
	}
	if err, ok := result.Value.(error); ok {
		return err
	}
	return core.E("HTTPStream.Close", "close failed", nil)
}
