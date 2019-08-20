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

	"github.com/satori/go.uuid"
)

type Registry struct {
	repository PipelineRepository
}

func NewRegistry(repository PipelineRepository) *Registry {
	return &Registry{repository}
}

func (r *Registry) SavePipeline(pipeline Pipeline, userId string) (id uuid.UUID) {
	// Create new uuid to use as pipeline id
	id = uuid.NewV4()
	pipeline.Id = id.String()
	pipeline.UserId = userId
	pipeline.CreatedAt = time.Now()
	pipeline.UpdatedAt = time.Now()
	r.repository.InsertPipeline(pipeline)
	return
}

func (r *Registry) GetPipelines(userId string, args map[string][]string) (pipelines []Pipeline) {
	pipelines = r.repository.All(userId, args)
	return
}

func (r *Registry) GetPipeline(id string, userId string) (pipeline Pipeline) {
	pipeline = r.repository.FindPipeline(id, userId)
	return
}

func (r *Registry) DeletePipeline(id string, userId string) Response {
	err := r.repository.DeletePipeline(id, userId)
	if err != nil {
		fmt.Println("Could not delete pipeline record: " + err.Error())
	}
	return Response{"OK"}
}
