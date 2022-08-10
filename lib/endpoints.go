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

	"github.com/golang-jwt/jwt"

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
	uuid, err := REGISTRY.SavePipeline(pipeReq, getUserId(req))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(PipelineResponse{uuid})
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func PutPipelineEndpoint(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var pipeReq Pipeline
	err := decoder.Decode(&pipeReq)
	if err != nil {
		fmt.Println("Could not decode Pipeline Request data." + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	uuid, err := REGISTRY.UpdatePipeline(pipeReq, getUserId(req))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(PipelineResponse{uuid})
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func GetPipelineEndpoint(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	pipe, err := REGISTRY.GetPipeline(vars["id"], getUserId(req))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(pipe)
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func DeletePipelineEndpoint(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	resp, err := REGISTRY.DeletePipeline(vars["id"], getUserId(req))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func GetPipelinesEndpoint(w http.ResponseWriter, req *http.Request) {
	args := req.URL.Query()
	pipe, err := REGISTRY.GetPipelines(getUserId(req), args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(pipe)
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func GetPipelinesAdminEndpoint(w http.ResponseWriter, req *http.Request) {
	args := req.URL.Query()
	pipe, err := REGISTRY.GetPipelinesAdmin(getUserId(req), args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(pipe)
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func DeletePipelineAdminEndpoint(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	resp, err := REGISTRY.DeletePipelineAdmin(vars["id"], getUserId(req))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		fmt.Println("Could not encode response data." + err.Error())
	}
}

func getUserId(req *http.Request) (userId string) {
	userId = req.Header.Get("X-UserId")
	if userId == "" {
		if userId == "" && req.Header.Get("Authorization") != "" {
			_, claims := parseJWTToken(req.Header.Get("Authorization")[7:])
			userId = claims.Sub
			if userId == "" {
				userId = "dummy"
			}
		}
	}
	return
}

func parseJWTToken(encodedToken string) (token *jwt.Token, claims Claims) {
	token, _ = jwt.ParseWithClaims(encodedToken, &claims, nil)
	return
}
