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

var REGISTRY = NewRegistry(NewMongoRepo())

func GetRootEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(Response{"OK"})
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func PostPipelineEndpoint(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var pipeReq Pipeline
	err := decoder.Decode(&pipeReq)
	if err != nil {
		fmt.Println("Could not decode Pipeline Request data." + err.Error())
	}
	uuid := REGISTRY.SavePipeline(pipeReq, getUserId(req))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(PipelineResponse{uuid})
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func GetPipelineEndpoint(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(REGISTRY.GetPipeline(vars["id"], getUserId(req)))
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func DeletePipelineEndpoint(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(REGISTRY.DeletePipeline(vars["id"], getUserId(req)))
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func GetPipelinesEndpoint(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	args := req.URL.Query()
	err := json.NewEncoder(w).Encode(REGISTRY.GetPipelines(getUserId(req), args))
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func getUserId(req *http.Request) (userId string) {
	userId = req.Header.Get("X-UserId")
	if userId == "" {
		userId = "admin"
	}
	return
}
