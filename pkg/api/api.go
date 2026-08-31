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
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/analytics-pipeline/pkg/config"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/db"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/service"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	gin_mw "github.com/SENERGY-Platform/gin-middleware"
	"github.com/SENERGY-Platform/gin-middleware/otelx"
	"github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"
	permV2Client "github.com/SENERGY-Platform/permissions-v2/pkg/client"
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
// @version {version}
// @description For the administration of analytics pipelines.
// @license.name Apache-2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /
func CreateServer(ctx context.Context, cfg *config.Config, perm permV2Client.Client) (r *gin.Engine, err error) {
	port := strconv.FormatInt(int64(cfg.ServerPort), 10)
	util.Logger.InfoContext(ctx, "Starting api server at port "+port)

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
	// First in the chain: it extracts the trace context and the baggage off the
	// request, which everything after it reads from — the access log, the handlers,
	// and the mongo spans otelmongo produces under the request's own trace.
	otelHandler, err := otelx.GinOpenTelemetry(ctx, ServiceName, cfg.OtelEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to set up OpenTelemetry: %w", err)
	}
	var middleware []gin.HandlerFunc
	middleware = append(
		middleware,
		otelHandler,
		// Directly after it, and it has to stay there: see DiscardBaggageErrors.
		DiscardBaggageErrors(),
		gin_mw.StructLoggerHandlerWithDefaultGenerators(
			util.Logger.With(attributes.LogRecordTypeKey, attributes.HttpAccessLogRecordTypeVal),
			attributes.Provider,
			[]string{HealthCheckPath},
			nil,
		),
	)
	middleware = append(middleware,
		requestid.New(requestid.WithCustomHeaderStrKey(HeaderRequestID)),
		gin_mw.ErrorHandler(util.GetStatusCode, ", "),
		gin_mw.StructRecoveryHandler(util.Logger, gin_mw.DefaultRecoveryFunc),
	)
	r.Use(middleware...)
	r.UseRawPath = true
	prefix := r.Group(cfg.URLPrefix)

	REGISTRY := service.NewRegistry(db.NewMongoRepo(), perm)
	err = REGISTRY.ValidateOperatorPermissions(ctx)
	if err != nil {
		return nil, err
	}
	setRoutes, err := routes.Set(*REGISTRY, prefix)
	if err != nil {
		return nil, err
	}
	for _, route := range setRoutes {
		util.Logger.DebugContext(ctx, "http route", attributes.MethodKey, route[0], attributes.PathKey, route[1])
	}

	prefix.Use(AuthMiddleware())
	setRoutes, err = routesAuth.Set(*REGISTRY, prefix)
	if err != nil {
		return nil, err
	}
	for _, route := range setRoutes {
		util.Logger.DebugContext(ctx, "http route", attributes.MethodKey, route[0], attributes.PathKey, route[1])
	}

	prefix.Use(AdminMiddleware())
	setRoutes, err = routesAdmin.Set(*REGISTRY, prefix)
	if err != nil {
		return nil, err
	}
	for _, route := range setRoutes {
		util.Logger.DebugContext(ctx, "http route", attributes.MethodKey, route[0], attributes.PathKey, route[1])
	}
	return r, nil
}

// DiscardBaggageErrors takes the errors the OpenTelemetry handler reported off the
// request and logs them instead.
//
// otelx reports a baggage value it cannot carry — one holding a space, a comma or a
// non-ASCII character — with gin's c.Error. gin_mw.ErrorHandler then turns anything
// in c.Errors into a response: it forces a 500 where the status was below 400, and
// appends the error text to the body. A user whose Keycloak username is "Jonah
// Windolph" would therefore get a 500 on every DELETE and a corrupted JSON body on
// every GET, for a log annotation that failed.
//
// **This handler has to sit immediately after the OpenTelemetry handler.** otelx adds
// those errors before it calls c.Next(), so at this point in the chain nothing else
// can have added one, which is what makes clearing the slice safe. Moved further
// down, it would discard a real handler's error.
//
// The proper fix belongs in gin-middleware, which should not use c.Error for
// something that is not a request error.
func DiscardBaggageErrors() gin.HandlerFunc {
	return func(gc *gin.Context) {
		if len(gc.Errors) > 0 {
			for _, reported := range gc.Errors {
				util.Logger.WarnContext(gc.Request.Context(),
					"could not put a value into the request baggage", "error", reported.Err)
			}
			gc.Errors = nil
		}
		gc.Next()
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(gc *gin.Context) {
		userId, err := getUserId(gc)
		if err != nil {
			util.Logger.ErrorContext(gc.Request.Context(), "could not get user id", "error", err)
			gc.AbortWithStatus(http.StatusUnauthorized)
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
			util.Logger.ErrorContext(gc.Request.Context(), "could not check admin role", "error", err)
			gc.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !admin {
			util.Logger.WarnContext(gc.Request.Context(), "unauthorized user tries to access admin api")
			gc.AbortWithStatus(http.StatusUnauthorized)
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
