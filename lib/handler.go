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
	"slices"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var REGISTRY = NewRegistry(NewMongoRepo())

func CreateServer() {
	port := GetEnv("SERVICE_API_PORT", "8000")
	fmt.Print("Starting Server at port " + port + "\n")
	DEBUG, err := strconv.ParseBool(GetEnv("DEBUG", "false"))
	if err != nil {
		log.Print("Error loading debug value")
		DEBUG = false
	}
	if !DEBUG {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS", "PUT"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	prefix := r.Group(GetEnv("ROUTE_PREFIX", ""))
	prefix.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	prefix.POST("/pipeline", func(c *gin.Context) {
		var request Pipeline
		if err := c.ShouldBindJSON(&request); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		uuid, err := REGISTRY.SavePipeline(request, getUserId(c))
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, PipelineResponse{uuid})
	})

	prefix.PUT("/pipeline", func(c *gin.Context) {
		var request Pipeline
		if err := c.ShouldBindJSON(&request); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		uuid, err := REGISTRY.UpdatePipeline(request, getUserId(c))
		if err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, PipelineResponse{uuid})
	})

	prefix.GET("/pipeline/:id", func(c *gin.Context) {
		id := c.Param("id")
		pipe, err := REGISTRY.GetPipeline(id, getUserId(c))
		if err != nil {
			log.Println(err)
			return
		}
		c.JSON(http.StatusOK, pipe)
	})

	prefix.DELETE("/pipeline/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := REGISTRY.DeletePipeline(id, getUserId(c))
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	prefix.GET("/pipeline", func(c *gin.Context) {
		args := c.Request.URL.Query()
		pipes, err := REGISTRY.GetPipelines(getUserId(c), args)
		if err != nil {
			log.Println(err)
			return
		}
		c.JSON(http.StatusOK, pipes)
	})

	prefix.GET("/admin/pipeline", func(c *gin.Context) {
		args := c.Request.URL.Query()
		pipes, err := REGISTRY.GetPipelinesAdmin(getUserId(c), args)
		if err != nil {
			log.Println(err)
			return
		}
		c.JSON(http.StatusOK, pipes)
	})

	prefix.DELETE("/admin/pipeline/:id", func(c *gin.Context) {
		id := c.Param("id")
		_, err := REGISTRY.DeletePipelineAdmin(id, getUserId(c))
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	if !DEBUG {
		err = r.Run(":" + port)
	} else {
		err = r.Run("127.0.0.1:" + port)
	}
	if err != nil {
		log.Printf("Starting api server failed: %s \n", err)
	} else {
		log.Println("Server stopped without error message")
	}
}

func getUserId(c *gin.Context) (userId string) {
	forUser := c.Query("for_user")
	if forUser != "" {

		roles := strings.Split(c.GetHeader("X-User-Roles"), ", ")
		if slices.Contains[[]string](roles, "admin") {
			return forUser
		}
	}

	userId = c.GetHeader("X-UserId")
	if userId == "" {
		if c.GetHeader("Authorization") != "" {
			claims, err := jwt.Parse(c.GetHeader("Authorization"))
			if err != nil {
				return
			}
			userId = claims.Sub
			if userId == "" {
				userId = "dummy"
			}
		}
	}
	return
}
