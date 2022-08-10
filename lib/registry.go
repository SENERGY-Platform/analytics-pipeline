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
	"time"

	"github.com/google/uuid"

	"github.com/Nerzal/gocloak/v5"
)

type Registry struct {
	repository PipelineRepository
}

func NewRegistry(repository PipelineRepository) *Registry {
	return &Registry{repository}
}

func (r *Registry) SavePipeline(pipeline Pipeline, userId string) (id uuid.UUID, err error) {
	// Create new uuid to use as pipeline id
	id = uuid.New()
	pipeline.Id = id.String()
	pipeline.UserId = userId
	pipeline.CreatedAt = time.Now()
	pipeline.UpdatedAt = time.Now()
	err = r.repository.InsertPipeline(pipeline)
	return
}

func (r *Registry) UpdatePipeline(pipeline Pipeline, userId string) (id uuid.UUID, err error) {
	oldPipeline, err := r.repository.FindPipeline(pipeline.Id, userId)
	if err != nil {
		return [16]byte{}, err
	}
	pipeline.CreatedAt = oldPipeline.CreatedAt
	pipeline.UpdatedAt = time.Now()
	pipeline.UserId = oldPipeline.UserId
	err = r.repository.UpdatePipeline(pipeline, userId)
	if err != nil {
		return [16]byte{}, err
	}
	return
}

func (r *Registry) GetPipelines(userId string, args map[string][]string) (pipelines []Pipeline, err error) {
	return r.repository.All(userId, false, args)
}

func (r *Registry) GetPipelinesAdmin(userId string, args map[string][]string) (pipelines []Pipeline, err error) {
	clientId := GetEnv("KEYCLOAK_CLIENT_ID", "test")
	clientSecret := GetEnv("KEYCLOAK_CLIENT_SECRET", "test")
	realm := GetEnv("KEYCLOAK_REALM", "test")

	client := gocloak.NewClient(GetEnv("KEYCLOAK_ADDRESS", "http://test"))
	token, err := client.LoginClient(clientId, clientSecret, realm)
	if err != nil {
		fmt.Println("Login failed:" + err.Error())
	}
	if token != nil {
		roles, _ := client.GetRealmRolesByUserID(token.AccessToken, realm, userId)
		if hasRole("admin", roles) {
			pipelines, err = r.repository.All(userId, true, args)
		}
	}
	return
}

func (r *Registry) DeletePipelineAdmin(id string, userId string) Response {
	clientId := GetEnv("KEYCLOAK_CLIENT_ID", "test")
	clientSecret := GetEnv("KEYCLOAK_CLIENT_SECRET", "test")
	realm := GetEnv("KEYCLOAK_REALM", "test")

	client := gocloak.NewClient(GetEnv("KEYCLOAK_ADDRESS", "http://test"))
	token, err := client.LoginClient(clientId, clientSecret, realm)
	if err != nil {
		fmt.Println("Login failed:" + err.Error())
	}

	if token != nil {
		roles, _ := client.GetRealmRolesByUserID(token.AccessToken, realm, userId)
		if hasRole("admin", roles) {
			err = r.repository.DeletePipeline(id, userId, true)
			if err != nil {
				fmt.Println("Could not delete pipeline record: " + err.Error())
			}
		}
	}
	return Response{"OK"}
}

func (r *Registry) GetPipeline(id string, userId string) (pipeline Pipeline, err error) {
	return r.repository.FindPipeline(id, userId)
}

func (r *Registry) DeletePipeline(id string, userId string) Response {
	err := r.repository.DeletePipeline(id, userId, false)
	if err != nil {
		fmt.Println("Could not delete pipeline record: " + err.Error())
	}
	return Response{"OK"}
}

func hasRole(test string, list []*gocloak.Role) bool {
	for _, role := range list {
		if *role.Name == test {
			return true
		}
	}
	return false
}
