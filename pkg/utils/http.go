package utils

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func DoRequest(method string, url string, headers map[string]string, query map[string]string, body io.Reader) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if len(query) > 0 {
		q := req.URL.Query()
		for key, value := range query {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	//fmt.Println(req.URL.String())
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return client.Do(req)
}

func UnmarshalJson[T any](resp *http.Response, target *T) error {
	d := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	return d.Decode(target)
}

func IsSuccessStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode <= 299
}

func FromCookiesString(cookies string) map[string]string {
	tokens := strings.Split(cookies, ";")
	cookiesMap := make(map[string]string)
	for _, token := range tokens {
		kv := strings.Split(token, "=")
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		cookiesMap[key] = value
	}
	return cookiesMap
}
