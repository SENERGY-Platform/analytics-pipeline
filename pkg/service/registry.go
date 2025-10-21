/*
 * Copyright 2025 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"time"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/db"
	"github.com/google/uuid"
)

type Registry struct {
	repository db.PipelineRepository
}

func NewRegistry(repository db.PipelineRepository) *Registry {
	return &Registry{repository}
}

func (r *Registry) SavePipeline(pipeline lib.Pipeline, userId string) (id uuid.UUID, err error) {
	// Create new uuid to use as pipeline id
	id = uuid.New()
	pipeline.Id = id.String()
	pipeline.UserId = userId
	pipeline.CreatedAt = time.Now()
	pipeline.UpdatedAt = time.Now()
	err = r.repository.InsertPipeline(pipeline)
	return
}

func (r *Registry) UpdatePipeline(pipeline lib.Pipeline, userId string) (id uuid.UUID, err error) {
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

func (r *Registry) GetPipelines(userId string, args map[string][]string) (pipelines lib.PipelinesResponse, err error) {
	return r.repository.All(userId, false, args)
}

func (r *Registry) GetPipelinesAdmin(userId string, args map[string][]string) (pipelines lib.PipelinesResponse, err error) {
	return r.repository.All(userId, true, args)
}

func (r *Registry) DeletePipelineAdmin(id string, userId string) (lib.Response, error) {
	err := r.repository.DeletePipeline(id, userId, true)
	if err != nil {
		return lib.Response{}, err
	}
	return lib.Response{Message: "OK"}, nil
}

func (r *Registry) GetPipeline(id string, userId string) (pipeline lib.Pipeline, err error) {
	return r.repository.FindPipeline(id, userId)
}

func (r *Registry) DeletePipeline(id string, userId string) (lib.Response, error) {
	err := r.repository.DeletePipeline(id, userId, false)
	return lib.Response{Message: "OK"}, err
}
