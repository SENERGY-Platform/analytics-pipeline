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
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/db"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	permV2Client "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	permV2Model "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/google/uuid"
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

// writeContext keeps the values of ctx — the trace and the baggage — but drops its
// cancellation.
//
// Every write here touches two stores that have to agree, mongo and permissions-v2,
// and a cancellation landing between them leaves them disagreeing: a pipeline
// without a permission resource reads back as forbidden, and a deleted one leaves an
// orphan resource behind. ValidateOperatorPermissions repairs both at the next
// start, but until then the pipeline is unusable to the user who just created it.
//
// Doing work nobody waits for any more is the cheaper failure. The read paths keep
// the request's context.
func writeContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}

func (r *Registry) ValidateOperatorPermissions(ctx context.Context) (err error) {
	util.Logger.DebugContext(ctx, "validate pipeline permissions")
	resp, err := r.GetPipelinesAdmin(ctx, "", nil)
	if err != nil {
		return
	}
	permResources, err, _ := r.perm.ListResourcesWithAdminPermissionContext(ctx, permV2Client.InternalAdminToken, PermV2InstanceTopic, permV2Client.ListOptions{})
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

		_, err, _ = r.perm.SetPermissionContext(ctx, permV2Client.InternalAdminToken, PermV2InstanceTopic, pipeline.Id, permissions)
		if err != nil {
			return
		}
	}
	permResourceIds := maps.Keys(permResourceMap)

	for permResouceId := range permResourceIds {
		if !slices.Contains(dbIds, permResouceId) {
			err, _ = r.perm.RemoveResourceContext(ctx, permV2Client.InternalAdminToken, PermV2InstanceTopic, permResouceId)
			if err != nil {
				return
			}
			util.Logger.DebugContext(ctx, fmt.Sprintf("%s exists only in permissions-v2, now deleted", permResouceId))
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

func (r *Registry) SavePipeline(ctx context.Context, pipeline lib.Pipeline, userId string) (id string, err error) {
	ctx = writeContext(ctx)
	// Create new uuid to use as pipeline id
	uid := uuid.New()
	id = uid.String()
	pipeline.Id = id
	pipeline.UserId = userId
	pipeline.CreatedAt = time.Now()
	pipeline.UpdatedAt = time.Now()
	err = r.repository.InsertPipeline(ctx, pipeline)
	if err != nil {
		return
	}
	permissions := permV2Client.ResourcePermissions{
		GroupPermissions: map[string]permV2Client.PermissionsMap{},
		UserPermissions:  map[string]permV2Client.PermissionsMap{},
		RolePermissions:  map[string]permV2Model.PermissionsMap{},
	}
	SetDefaultPermissions(pipeline, permissions)
	_, err, _ = r.perm.SetPermissionContext(ctx, permV2Client.InternalAdminToken, PermV2InstanceTopic, pipeline.Id, permissions)
	return
}

func (r *Registry) UpdatePipeline(ctx context.Context, pipeline lib.Pipeline, userId string, auth string) (id string, err error) {
	ctx = writeContext(ctx)
	ok, err, _ := r.perm.CheckPermissionContext(ctx, auth, PermV2InstanceTopic, pipeline.Id, permV2Client.Write)
	if err != nil {
		return
	}
	if !ok {
		return id, lib.NewForbiddenError(errors.New(MessageMissingRights))
	}

	oldPipeline, err := r.repository.FindPipeline(ctx, pipeline.Id, userId)
	if err != nil {
		return id, err
	}
	pipeline.CreatedAt = oldPipeline.CreatedAt
	pipeline.UpdatedAt = time.Now()
	pipeline.UserId = oldPipeline.UserId
	err = r.repository.UpdatePipeline(ctx, pipeline, userId)
	if err != nil {
		return id, err
	}
	return
}

func (r *Registry) GetPipelines(ctx context.Context, userId string, args map[string][]string, auth string) (pipelines lib.PipelinesResponse, err error) {
	stringIds, err, _ := r.perm.ListAccessibleResourceIdsContext(ctx, auth, PermV2InstanceTopic, permV2Client.ListOptions{}, permV2Client.Read)
	return r.repository.All(ctx, userId, false, args, stringIds)
}

func (r *Registry) GetPipelinesAdmin(ctx context.Context, userId string, args map[string][]string) (pipelines lib.PipelinesResponse, err error) {
	return r.repository.All(ctx, userId, true, args, []string{})
}

func (r *Registry) GetPipelineUserCount(ctx context.Context, userId string, args map[string][]string) (statistics []lib.PipelineUserCount, err error) {
	return r.repository.PipelineUserCount(ctx, userId, true, args)
}

func (r *Registry) GetOperatorUsage(ctx context.Context, userId string, args map[string][]string) (statistics []lib.OperatorUsage, err error) {
	return r.repository.OperatorUsage(ctx, userId, true, args)
}

func (r *Registry) GetFlowUsage(ctx context.Context) (statistics []lib.FlowUsage, err error) {
	return r.repository.FlowUsage(ctx, "")
}

func (r *Registry) GetFlowUsageById(ctx context.Context, id string) (statistics *lib.FlowUsage, err error) {
	resp, err := r.repository.FlowUsage(ctx, id)
	if err != nil {
		return
	}
	if len(resp) == 0 {
		return
	}
	return &resp[0], nil
}

func (r *Registry) DeletePipelineAdmin(ctx context.Context, id string, userId string) (err error) {
	ctx = writeContext(ctx)
	err = r.repository.DeletePipeline(ctx, id, userId, true)
	if err != nil {
		return
	}
	err, _ = r.perm.RemoveResourceContext(ctx, permV2Client.InternalAdminToken, PermV2InstanceTopic, id)
	return err
}

func (r *Registry) GetPipeline(ctx context.Context, id string, userId string, auth string) (pipeline lib.Pipeline, err error) {
	ok, err, _ := r.perm.CheckPermissionContext(ctx, auth, PermV2InstanceTopic, id, permV2Client.Read)
	if err != nil {
		return
	}
	if !ok {
		return pipeline, lib.NewForbiddenError(errors.New(MessageMissingRights))
	}
	return r.repository.FindPipeline(ctx, id, userId)
}

func (r *Registry) DeletePipeline(ctx context.Context, id string, userId string, auth string) (err error) {
	ctx = writeContext(ctx)
	ok, err, _ := r.perm.CheckPermissionContext(ctx, auth, PermV2InstanceTopic, id, permV2Client.Administrate)
	if err != nil {
		return
	}
	if !ok {
		return lib.NewForbiddenError(errors.New(MessageMissingRights))
	}
	err = r.repository.DeletePipeline(ctx, id, userId, false)
	if err != nil {
		return
	}
	err, _ = r.perm.RemoveResourceContext(ctx, permV2Client.InternalAdminToken, PermV2InstanceTopic, id)
	return
}
