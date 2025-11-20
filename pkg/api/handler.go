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

package api

import (
	"errors"
	"net/http"
	"os"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/service"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	"github.com/gin-gonic/gin"
)

// postPipeline returns a handler function for the "/pipeline" endpoint that saves a pipeline
// @Summary Save a pipeline
// @Description Saves a pipeline given a pipeline request
// @Tags pipelines
// @Accept json
// @Produce json
// @Param request body lib.Pipeline true "Pipeline request"
// @Success 200 {object} lib.PipelineResponse
// @Failure 400 {string} MessageBadInput
// @Failure 401
// @Failure 500 {string} MessageSomethingWrong
// @Router /pipeline [post]
// @Security Bearer
func postPipeline(registry service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodPost, PipelinePath, func(c *gin.Context) {
		var request lib.Pipeline
		if err := c.ShouldBindJSON(&request); err != nil {
			util.Logger.Error("error parsing request", "error", err, "method", "POST", "path", PipelinePath)
			_ = c.Error(lib.NewInputError(errors.New(MessageBadInput)))
			return
		}
		uuid, err := registry.SavePipeline(request, c.GetString(UserIdKey))
		if err != nil {
			util.Logger.Error("could not get save pipeline", "error", err, "method", "POST", "path", PipelinePath)
			_ = c.Error(handleError(err))
			return
		}
		c.JSON(http.StatusOK, lib.PipelineResponse{Id: uuid})
	}
}

// putPipeline returns a handler function for the "/pipeline" endpoint that updates a pipeline
// @Summary Update a pipeline
// @Description Updates a pipeline given a pipeline request
// @Tags pipelines
// @Accept json
// @Produce json
// @Param request body lib.Pipeline true "Pipeline request"
// @Success 200 {object} lib.PipelineResponse
// @Failure 400 {string} MessageBadInput
// @Failure 401
// @Failure 403 {string} MessageForbidden
// @Failure 404 {string} MessageNotFound
// @Failure 500 {string} MessageSomethingWrong
// @Router /pipeline [put]
// @Security Bearer
func putPipeline(registry service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodPut, PipelinePath, func(c *gin.Context) {
		var request lib.Pipeline
		if err := c.ShouldBindJSON(&request); err != nil {
			util.Logger.Error("error parsing request", "error", err, "method", "POST", "path", PipelinePath)
			_ = c.Error(lib.NewInputError(errors.New(MessageBadInput)))
			return
		}
		uuid, err := registry.UpdatePipeline(request, c.GetString(UserIdKey), c.GetHeader(HeaderAuthorization))
		if err != nil {
			util.Logger.Error("could not get save pipeline", "error", err, "method", "POST", "path", PipelinePath)
			_ = c.Error(handleError(err))
			return
		}
		c.JSON(http.StatusOK, lib.PipelineResponse{Id: uuid})
	}
}

// getPipeline returns a handler function for the "/pipeline/:id" endpoint that retrieves a pipeline
// @Summary Retrieve a pipeline
// @Description Retrieves a pipeline given a pipeline ID
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path string true "Pipeline ID"
// @Success 200 {object} lib.Pipeline
// @Failure 401
// @Failure 403 {string} MessageForbidden
// @Failure 404 {string} MessageNotFound
// @Failure 500 {string} MessageSomethingWrong
// @Router /pipeline/:id [get]
// @Security Bearer
func getPipeline(registry service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "/pipeline/:id", func(c *gin.Context) {
		id := c.Param("id")
		pipe, err := registry.GetPipeline(id, c.GetString(UserIdKey), c.GetHeader(HeaderAuthorization))
		if err != nil {
			util.Logger.Error("could not get pipeline", "error", err, "method", "GET", "path", "/pipeline/"+id)
			_ = c.Error(handleError(err))
			return
		}
		c.JSON(http.StatusOK, pipe)
	}
}

// deletePipeline returns a handler function for the "/pipeline/:id" endpoint that deletes a pipeline
// @Summary Delete a pipeline
// @Description Deletes a pipeline given a pipeline ID
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path string true "Pipeline ID"
// @Success 200
// @Failure 401
// @Failure 403 {string} MessageForbidden
// @Failure 404 {string} MessageNotFound
// @Failure 500 {string} MessageSomethingWrong
// @Router /pipeline/:id [delete]
// @Security Bearer
func deletePipeline(registry service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, "/pipeline/:id", func(c *gin.Context) {
		id := c.Param("id")
		err := registry.DeletePipeline(id, c.GetString(UserIdKey), c.GetHeader(HeaderAuthorization))
		if err != nil {
			util.Logger.Error("could not delete pipeline", "error", err, "method", "DELETE", "path", "/pipeline/"+id, "userId", c.GetString(UserIdKey))
			_ = c.Error(handleError(err))
			return
		}
		c.Status(http.StatusOK)
	}
}

// getPipelines returns a handler function for the "/pipeline" endpoint that retrieves a list of pipelines
// @Summary Retrieve a list of pipelines
// @Description Retrieves a list of pipelines given a set of query parameters
// @Tags pipelines
// @Accept json
// @Produce json
// @Param query query string false "Query parameters"
// @Success 200 {object} []lib.Pipeline
// @Failure 401
// @Failure 500 {string} MessageSomethingWrong
// @Router /pipeline [get]
// @Security Bearer
func getPipelines(registry service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodGet, PipelinePath, func(c *gin.Context) {
		args := c.Request.URL.Query()
		pipes, err := registry.GetPipelines(c.GetString(UserIdKey), args, c.GetHeader(HeaderAuthorization))
		if err != nil {
			util.Logger.Error("could not get pipelines", "error", err, "method", "GET", "path", PipelinePath)
			_ = c.Error(handleError(err))
			return
		}
		c.JSON(http.StatusOK, pipes)
	}
}

func getPipelinesAdmin(registry service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "/admin/pipeline", func(c *gin.Context) {
		args := c.Request.URL.Query()
		pipes, err := registry.GetPipelinesAdmin(c.GetString(UserIdKey), args)
		if err != nil {
			util.Logger.Error("could not get pipelines for admin", "error", err, "method", "GET", "path", "/admin/pipeline/")
			_ = c.Error(handleError(err))
			return
		}
		c.JSON(http.StatusOK, pipes)
	}
}

func getPipelineUserCountAdmin(registry service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "/admin/pipeline/statistics/usercount", func(c *gin.Context) {
		args := c.Request.URL.Query()
		statistics, err := registry.GetPipelineUserCount(c.GetString(UserIdKey), args)
		if err != nil {
			util.Logger.Error("could not get PipelineUserCount statistics for admin", "error", err, "method", "GET", "path", "/admin/pipeline/statistics/usercount")
			_ = c.Error(handleError(err))
			return
		}
		c.JSON(http.StatusOK, statistics)
	}
}

func getOperatorUsageAdmin(registry service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "/admin/pipeline/statistics/operatorusage", func(c *gin.Context) {
		args := c.Request.URL.Query()
		statistics, err := registry.GetOperatorUsage(c.GetString(UserIdKey), args)
		if err != nil {
			util.Logger.Error("could not get OperatorUsage statistics for admin", "error", err, "method", "GET", "path", "/admin/pipeline/statistics/operatorusage")
			_ = c.Error(handleError(err))
			return
		}
		c.JSON(http.StatusOK, statistics)
	}
}

func deletePipelineAdmin(registry service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, "/admin/pipeline/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := registry.DeletePipelineAdmin(id, c.GetString(UserIdKey))
		if err != nil {
			util.Logger.Error("could not delete pipeline for admin", "error", err, "method", "DELETE", "path", "/admin/pipeline/"+id)
			_ = c.Error(handleError(err))
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func getHealthCheckH(_ service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodGet, HealthCheckPath, func(c *gin.Context) {
		c.Status(http.StatusOK)
	}
}

func getSwaggerDocH(_ service.Registry) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "/doc", func(gc *gin.Context) {
		if _, err := os.Stat("docs/swagger.json"); err != nil {
			_ = gc.Error(handleError(err))
			return
		}
		gc.Header("Content-Type", gin.MIMEJSON)
		gc.File("docs/swagger.json")
	}
}
