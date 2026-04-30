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

func (c *ConcurrencyLimit) UnmarshalYAML(value *yaml.Node) error {
	var n int
	if err := value.Decode(&n); err == nil {
		c.Total = n
		return nil
	}
	var m map[string]int
	if err := value.Decode(&m); err != nil {
		return err
	}
	c.Total = m["total"]
	c.Models = make(map[string]int)
	for key, entry := range m {
		if key != "total" {
			c.Models[key] = entry
		}
	}
	return nil
}

type HTTPStream struct {
	Client   *http.Client
	URL      string
	Token    string
	Method   string
	Response []byte
}

func (s *HTTPStream) Send(data []byte) error {
	request, err := http.NewRequestWithContext(context.Background(), s.Method, s.URL, core.NewReader(string(data)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if s.Token != "" {
		request.Header.Set("Authorization", core.Concat("token ", s.Token))
	}

	response, err := s.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	readResult := core.ReadAll(response.Body)
	if !readResult.OK {
		err, _ := readResult.Value.(error)
		return core.E("httpStream.Send", "failed to read response", err)
	}
	s.Response = []byte(readResult.Value.(string))
	return nil
}

func (s *HTTPStream) Receive() ([]byte, error) {
	return s.Response, nil
}

func (s *HTTPStream) Close() error {
	return nil
}
