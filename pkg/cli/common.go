package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"pangolin/pkg/utils"
)

func (c *jboxCli) postRequest(url string, query map[string]string, body io.Reader) (*http.Response, error) {
	return utils.DoRequest(http.MethodPost, url, c.headers, query, body)
}

func (c *jboxCli) postJson(url string, query map[string]string, body any) (*http.Response, error) {
	headers := map[string]string{}
	for k, v := range c.headers {
		headers[k] = v
	}
	headers["Content-Type"] = "application/json"
	marshalled, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return utils.DoRequest(http.MethodPost, url, headers, query, bytes.NewReader(marshalled))
}

func (c *jboxCli) getRequest(url string, query map[string]string) (*http.Response, error) {
	return utils.DoRequest(http.MethodGet, url, c.headers, query, nil)
}

func (c *jboxCli) putRequest(url string, headers map[string]string, query map[string]string, body io.Reader) (*http.Response, error) {
	if headers != nil {
		for k, v := range c.headers {
			headers[k] = v
		}
	} else {
		headers = c.headers
	}
	return utils.DoRequest(http.MethodPut, url, headers, query, body)
}
