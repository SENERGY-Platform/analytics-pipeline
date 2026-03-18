/*
 * Copyright 2023 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseUrl string
}

func NewClient(baseUrl string) *Client {
	return &Client{baseUrl: baseUrl}
}

func do[T any](req *http.Request, token string, userId string) (result T, err error, code int) {
	req.Header.Set("Authorization", withBearer(token))
	req.Header.Set("X-UserId", userId)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	code = resp.StatusCode

	if code >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf(
			"unexpected status code %d: %s",
			code,
			strings.TrimSpace(string(body)),
		), code
	}

	if resp.ContentLength == 0 {
		return result, nil, code
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, err, code
	}
	return
}

func withBearer(token string) string {
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token
	}
	return "Bearer " + token
}
