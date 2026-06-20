package utils

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type ProgressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	onProgress func(downloaded int64, total int64)
}

func NewProgressReader(r io.Reader, total int64, onProgress func(int64, int64)) *ProgressReader {
	return &ProgressReader{
		reader:     r,
		total:      total,
		onProgress: onProgress,
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	if pr.onProgress != nil {
		pr.onProgress(pr.read, pr.total)
	}
	return n, err
}

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
