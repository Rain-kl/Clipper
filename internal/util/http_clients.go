/*
Copyright 2025-2026 linux.do
Modified by Arctel.net, 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// IsLocalhost 检查 URL 是否为 localhost
func IsLocalhost(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	hostname := u.Hostname()
	return hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}

// 配置HTTP客户端 使用 otelhttp 自动注入 trace span
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: otelhttp.NewTransport(&http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     60 * time.Second,
	}),
}

func SetHTTPClient(c *http.Client) {
	httpClient = c
}

func Request(ctx context.Context, method, url string, body io.Reader, headers, cookies map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf(errCreateHTTPRequestFailed, err)
	}

	for key, value := range cookies {
		req.AddCookie(&http.Cookie{Name: key, Value: value})
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(errHTTPRequestFailed, url, err)
	}

	return resp, nil
}
