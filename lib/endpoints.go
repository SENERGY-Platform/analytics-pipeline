/*
 * Copyright 2018 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lib

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type Endpoint struct {
}

func NewEndpoint() *Endpoint {
	return &Endpoint{}
}

func (e *Endpoint) getRootEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(Response{"OK"})
}

func (e *Endpoint) postPipelineEndpoint(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var pipeReq Pipeline
	err := decoder.Decode(&pipeReq)
	if err != nil {
		fmt.Println("Could not decode Pipeline Request data." + err.Error())
	}
	uuid := savePipeline(pipeReq, getUserId(req))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(PipelineResponse{uuid})
}

func (e *Endpoint) getPipelineEndpoint(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(getPipeline(vars["id"], getUserId(req)))
}

func (e *Endpoint) deletePipelineEndpoint(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(deletePipeline(vars["id"], getUserId(req)))
}

func (e *Endpoint) getPipelinesEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	args := req.URL.Query()
	json.NewEncoder(w).Encode(getPipelines(getUserId(req), args))
}

func getUserId(req *http.Request) (userId string) {
	userId = req.Header.Get("X-UserId")
	if userId == "" {
		userId = "admin"
	}
	fmt.Println("UserID: " + userId)
	return
}
