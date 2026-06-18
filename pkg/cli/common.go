package cli

import (
	"io"
	"net/http"
	"pangolin/pkg/utils"
)

func (c *jboxCli) postRequest(url string, query map[string]string, body io.Reader) (*http.Response, error) {
	return utils.DoRequest(http.MethodPost, url, c.headers, query, body)
}

func (c *jboxCli) getRequest(url string, query map[string]string) (*http.Response, error) {
	return utils.DoRequest(http.MethodGet, url, c.headers, query, nil)
}
