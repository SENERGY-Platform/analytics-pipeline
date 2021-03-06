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
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func CreateServer() {
	port := GetEnv("API_PORT", "8000")
	fmt.Print("Starting Server at port " + port + "\n")
	router := mux.NewRouter()
	router.HandleFunc("/", GetRootEndpoint).Methods("GET")
	router.HandleFunc("/pipeline", PostPipelineEndpoint).Methods("POST")
	router.HandleFunc("/pipeline", PutPipelineEndpoint).Methods("PUT")
	router.HandleFunc("/pipeline/{id}", GetPipelineEndpoint).Methods("GET")
	router.HandleFunc("/pipeline/{id}", DeletePipelineEndpoint).Methods("DELETE")
	router.HandleFunc("/pipeline", GetPipelinesEndpoint).Methods("GET")
	router.HandleFunc("/admin/pipeline", GetPipelinesAdminEndpoint).Methods("GET")
	router.HandleFunc("/admin/pipeline/{id}", DeletePipelineAdminEndpoint).Methods("DELETE")
	c := cors.New(
		cors.Options{
			AllowedHeaders: []string{"Content-Type", "Authorization"},
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "PUT", "POST", "DELETE", "OPTIONS"},
		})
	handler := c.Handler(router)
	logger := NewLogger(handler, "CALL")
	log.Fatal(http.ListenAndServe(GetEnv("SERVERNAME", "")+":"+port, logger))
}
