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
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"github.com/google/uuid"
)

func (c *Client) SavePipeline(token string, userId string, pipeline lib.Pipeline) (id uuid.UUID, err error, code int) {
	b, err := json.Marshal(pipeline)
	if err != nil {
		return id, err, http.StatusBadRequest
	}
	req, err := http.NewRequest(http.MethodPost, c.baseUrl+"/pipeline", bytes.NewBuffer(b))
	return do[uuid.UUID](req, token, userId)
}

func (c *Client) UpdatePipeline(token string, userId string, pipeline lib.Pipeline) (id uuid.UUID, err error, code int) {
	b, err := json.Marshal(pipeline)
	if err != nil {
		return id, err, http.StatusBadRequest
	}
	req, err := http.NewRequest(http.MethodPut, c.baseUrl+"/pipeline", bytes.NewBuffer(b))
	return do[uuid.UUID](req, token, userId)
}
func (c *Client) GetPipelines(token string, userId string, limit int, offset int, order string, asc bool) (pipelines lib.PipelinesResponse, err error, code int) {
	url := c.baseUrl + "/pipeline?limit=" + strconv.Itoa(limit) + "&offset=" + strconv.Itoa(offset)
	if order != "" {
		url += "order=" + order + "."
		if asc {
			url += "asc"
		} else {
			url += "desc"
		}
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	return do[lib.PipelinesResponse](req, token, userId)
}

func (c *Client) GetPipelinesAdmin(token string, userId string, limit int, offset int, order string, asc bool) (pipelines lib.PipelinesResponse, err error, code int) {
	url := c.baseUrl + "/admin/pipeline?limit=" + strconv.Itoa(limit) + "&offset=" + strconv.Itoa(offset)
	if order != "" {
		url += "order=" + order + "."
		if asc {
			url += "asc"
		} else {
			url += "desc"
		}
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	return do[lib.PipelinesResponse](req, token, userId)
}

func (c *Client) DeletePipelineAdmin(token string, userId string, id string) (resp lib.Response, err error, code int) {
	req, err := http.NewRequest(http.MethodDelete, c.baseUrl+"/admin/pipeline/"+id, nil)
	return do[lib.Response](req, token, userId)
}

func (c *Client) GetPipeline(token string, userId string, id string) (pipeline lib.Pipeline, err error, code int) {
	req, err := http.NewRequest(http.MethodGet, c.baseUrl+"/pipeline/"+id, nil)
	return do[lib.Pipeline](req, token, userId)
}

func (c *Client) DeletePipeline(token string, userId string, id string) (resp lib.Response, err error, code int) {
	req, err := http.NewRequest(http.MethodDelete, c.baseUrl+"/pipeline/"+id, nil)
	return do[lib.Response](req, token, userId)
}
