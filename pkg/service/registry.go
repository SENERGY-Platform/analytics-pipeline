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
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/db"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	"github.com/google/uuid"

	permV2Client "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	permV2Model "github.com/SENERGY-Platform/permissions-v2/pkg/model"
)

type Registry struct {
	repository db.PipelineRepository
	perm       permV2Client.Client
}

func NewRegistry(repository db.PipelineRepository, perm permV2Client.Client) *Registry {
	_, err, _ := perm.SetTopic(permV2Client.InternalAdminToken, permV2Client.Topic{
		Id: PermV2InstanceTopic,
		DefaultPermissions: permV2Client.ResourcePermissions{
			RolePermissions: map[string]permV2Model.PermissionsMap{
				"admin": {
					Read:         true,
					Write:        true,
					Execute:      true,
					Administrate: true,
				},
			},
		},
	})
	if err != nil {
		return nil
	}
	return &Registry{repository, perm}
}

func (r *Registry) ValidateOperatorPermissions() (err error) {
	util.Logger.Debug("validate pipeline permissions")
	resp, err := r.GetPipelinesAdmin("", nil)
	if err != nil {
		return
	}
	permResources, err, _ := r.perm.ListResourcesWithAdminPermission(permV2Client.InternalAdminToken, PermV2InstanceTopic, permV2Client.ListOptions{})
	if err != nil {
		return
	}
	permResourceMap := map[string]permV2Client.Resource{}
	for _, permResource := range permResources {
		permResourceMap[permResource.Id] = permResource
	}

	dbIds := []string{}
	for _, pipeline := range resp.Data {
		permissions := permV2Client.ResourcePermissions{
			UserPermissions:  map[string]permV2Client.PermissionsMap{},
			GroupPermissions: map[string]permV2Client.PermissionsMap{},
			RolePermissions:  map[string]permV2Model.PermissionsMap{},
		}
		dbIds = append(dbIds, pipeline.Id)
		resource, ok := permResourceMap[pipeline.Id]
		if ok {
			permissions.UserPermissions = resource.ResourcePermissions.UserPermissions
			permissions.GroupPermissions = resource.GroupPermissions
			permissions.RolePermissions = resource.ResourcePermissions.RolePermissions
		}
		SetDefaultPermissions(pipeline, permissions)

		_, err, _ = r.perm.SetPermission(permV2Client.InternalAdminToken, PermV2InstanceTopic, pipeline.Id, permissions)
		if err != nil {
			return
		}
	}
	permResourceIds := maps.Keys(permResourceMap)

	for permResouceId := range permResourceIds {
		if !slices.Contains(dbIds, permResouceId) {
			err, _ = r.perm.RemoveResource(permV2Client.InternalAdminToken, PermV2InstanceTopic, permResouceId)
			if err != nil {
				return
			}
			util.Logger.Debug(fmt.Sprintf("%s exists only in permissions-v2, now deleted", permResouceId))
		}
	}
	return
}

func SetDefaultPermissions(instance lib.Pipeline, permissions permV2Client.ResourcePermissions) {
	permissions.UserPermissions[instance.UserId] = permV2Client.PermissionsMap{
		Read:         true,
		Write:        true,
		Execute:      true,
		Administrate: true,
	}
}

func (r *Registry) SavePipeline(pipeline lib.Pipeline, userId string) (id uuid.UUID, err error) {
	// Create new uuid to use as pipeline id
	id = uuid.New()
	pipeline.Id = id.String()
	pipeline.UserId = userId
	pipeline.CreatedAt = time.Now()
	pipeline.UpdatedAt = time.Now()
	err = r.repository.InsertPipeline(pipeline)
	if err != nil {
		return
	}
	permissions := permV2Client.ResourcePermissions{
		GroupPermissions: map[string]permV2Client.PermissionsMap{},
		UserPermissions:  map[string]permV2Client.PermissionsMap{},
		RolePermissions:  map[string]permV2Model.PermissionsMap{},
	}
	SetDefaultPermissions(pipeline, permissions)
	_, err, _ = r.perm.SetPermission(permV2Client.InternalAdminToken, PermV2InstanceTopic, pipeline.Id, permissions)
	return
}

func (r *Registry) UpdatePipeline(pipeline lib.Pipeline, userId string, auth string) (id uuid.UUID, err error) {
	ok, err, _ := r.perm.CheckPermission(auth, PermV2InstanceTopic, pipeline.Id, permV2Client.Write)
	if err != nil {
		return
	}
	if !ok {
		return id, errors.New(MessageMissingRights)
	}

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

func (r *Registry) GetPipelines(userId string, args map[string][]string, auth string) (pipelines lib.PipelinesResponse, err error) {
	stringIds, err, _ := r.perm.ListAccessibleResourceIds(auth, PermV2InstanceTopic, permV2Client.ListOptions{}, permV2Client.Read)
	return r.repository.All(userId, false, args, stringIds)
}

func (r *Registry) GetPipelinesAdmin(userId string, args map[string][]string) (pipelines lib.PipelinesResponse, err error) {
	return r.repository.All(userId, true, args, []string{})
}

func (r *Registry) GetPipelineUserCount(userId string, args map[string][]string) (statistics []lib.PipelineUserCount, err error) {
	return r.repository.PipelineUserCount(userId, true, args)
}

func (r *Registry) GetOperatorUsage(userId string, args map[string][]string) (statistics []lib.OperatorUsage, err error) {
	return r.repository.OperatorUsage(userId, true, args)
}

func (r *Registry) DeletePipelineAdmin(id string, userId string) (resp lib.Response, err error) {
	err = r.repository.DeletePipeline(id, userId, true)
	if err != nil {
		return
	}
	err, _ = r.perm.RemoveResource(permV2Client.InternalAdminToken, PermV2InstanceTopic, id)
	return lib.Response{Message: "OK"}, nil
}

func (r *Registry) GetPipeline(id string, userId string, auth string) (pipeline lib.Pipeline, err error) {
	ok, err, _ := r.perm.CheckPermission(auth, PermV2InstanceTopic, id, permV2Client.Read)
	if err != nil {
		return
	}
	if !ok {
		return pipeline, errors.New(MessageMissingRights)
	}
	return r.repository.FindPipeline(id, userId)
}

func (r *Registry) DeletePipeline(id string, userId string, auth string) (err error) {
	ok, err, _ := r.perm.CheckPermission(auth, PermV2InstanceTopic, id, permV2Client.Administrate)
	if err != nil {
		return
	}
	if !ok {
		return errors.New(MessageMissingRights)
	}
	err = r.repository.DeletePipeline(id, userId, false)
	if err != nil {
		return
	}
	err, _ = r.perm.RemoveResource(permV2Client.InternalAdminToken, PermV2InstanceTopic, id)
	return
}
