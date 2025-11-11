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
	"slices"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/analytics-pipeline/pkg/config"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/db"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/service"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	gin_mw "github.com/SENERGY-Platform/gin-middleware"
	"github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"
	"github.com/SENERGY-Platform/service-commons/pkg/jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

// CreateServer creates a new gin.Engine instance and configures it according to the given config.
// It sets up the middleware for logging, recovery, authentication, and request ID tracking.
// It also sets up the routes for the API using the given registry.
// The server is started at the port specified in the config.
// @title Analytics-Pipeline API
// @version 0.0.21
// @description For the administration of analytics pipelines.
// @license.name Apache-2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /
func CreateServer(cfg *config.Config) (r *gin.Engine, err error) {
	port := strconv.FormatInt(int64(cfg.ServerPort), 10)
	util.Logger.Info("Starting api server at port " + port)

	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	r = gin.New()
	r.RedirectTrailingSlash = false
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS", "PUT"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	var middleware []gin.HandlerFunc
	middleware = append(
		middleware,
		gin_mw.StructLoggerHandlerWithDefaultGenerators(
			util.Logger.With(attributes.LogRecordTypeKey, attributes.HttpAccessLogRecordTypeVal),
			attributes.Provider,
			[]string{HealthCheckPath},
			nil,
		),
	)
	middleware = append(middleware,
		requestid.New(requestid.WithCustomHeaderStrKey(HeaderRequestID)),
		gin_mw.ErrorHandler(func(err error) int {
			return 0
		}, ", "),
		gin_mw.StructRecoveryHandler(util.Logger, gin_mw.DefaultRecoveryFunc),
	)
	r.Use(middleware...)
	r.UseRawPath = true
	prefix := r.Group(cfg.URLPrefix)
	REGISTRY := service.NewRegistry(db.NewMongoRepo())
	setRoutes, err := routes.Set(*REGISTRY, prefix)
	if err != nil {
		return nil, err
	}
	for _, route := range setRoutes {
		util.Logger.Debug("http route", attributes.MethodKey, route[0], attributes.PathKey, route[1])
	}

	prefix.Use(AuthMiddleware())
	setRoutes, err = routesAuth.Set(*REGISTRY, prefix)
	if err != nil {
		return nil, err
	}
	for _, route := range setRoutes {
		util.Logger.Debug("http route", attributes.MethodKey, route[0], attributes.PathKey, route[1])
	}

	prefix.Use(AdminMiddleware())
	setRoutes, err = routesAdmin.Set(*REGISTRY, prefix)
	if err != nil {
		return nil, err
	}
	for _, route := range setRoutes {
		util.Logger.Debug("http route", attributes.MethodKey, route[0], attributes.PathKey, route[1])
	}
	return r, nil
}

func AuthMiddleware() gin.HandlerFunc {
	return func(gc *gin.Context) {
		userId, err := getUserId(gc)
		if err != nil {
			util.Logger.Error("could not get user id", "error", err)
			gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		gc.Set(UserIdKey, userId)
		gc.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(gc *gin.Context) {
		admin, err := isAdmin(gc)
		if err != nil {
			util.Logger.Error("could not check admin role", "error", err)
			gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if !admin {
			util.Logger.Warn("unauthorized user tries to access admin api")
			gc.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		gc.Set(AdminKey, true)
		gc.Next()
	}
}

func isAdmin(c *gin.Context) (result bool, err error) {
	rolesHeader := c.GetHeader("X-User-Roles")
	if rolesHeader != "" {
		roles := strings.Split(rolesHeader, ", ")
		if slices.Contains[[]string](roles, "admin") {
			return true, nil
		}
		return false, nil
	}
	if c.GetHeader("Authorization") != "" {
		var claims jwt.Token
		claims, err = jwt.Parse(c.GetHeader("Authorization"))
		if err != nil {
			return
		}
		return claims.IsAdmin(), nil
	}
	return false, nil
}

func getUserId(c *gin.Context) (userId string, err error) {
	forUser := c.Query("for_user")
	if forUser != "" {
		roles := strings.Split(c.GetHeader("X-User-Roles"), ", ")
		if slices.Contains[[]string](roles, "admin") {
			return forUser, nil
		}
	}

	userId = c.GetHeader("X-UserId")
	if userId == "" {
		if c.GetHeader("Authorization") != "" {
			var claims jwt.Token
			claims, err = jwt.Parse(c.GetHeader("Authorization"))
			if err != nil {
				return
			}
			userId = claims.Sub
		} else {
			err = errors.New("missing authorization and x-userid header")
		}
	}
	return
}
