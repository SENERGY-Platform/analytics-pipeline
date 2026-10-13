/*
 * Copyright 2026 InfAI (CC SES)
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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/db"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/service"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	gin_mw "github.com/SENERGY-Platform/gin-middleware"
	permV2Client "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/gin-gonic/gin"
)

// noopPerm accepts the topic NewRegistry registers; the usage endpoint needs nothing else.
type noopPerm struct {
	permV2Client.Client
}

func (noopPerm) SetTopic(_ string, topic permV2Client.Topic) (permV2Client.Topic, error, int) {
	return topic, nil, http.StatusOK
}

// usageRepo records the arguments of OperatorPipelineUsage and returns a fixed result.
type usageRepo struct {
	db.PipelineRepository
	calls                   int
	operatorId, image, user string
	usage                   lib.OperatorPipelineUsage
	err                     error
}

func (r *usageRepo) OperatorPipelineUsage(_ context.Context, operatorId string, imageId string, userId string) (lib.OperatorPipelineUsage, error) {
	r.calls++
	r.operatorId, r.image, r.user = operatorId, imageId, userId
	return r.usage, r.err
}

// newUsageRouter builds the authenticated route group the way CreateServer does.
func newUsageRouter(t *testing.T, repo *usageRepo) *gin.Engine {
	t.Helper()
	util.InitStructLogger("error")
	gin.SetMode(gin.TestMode)
	registry := service.NewRegistry(repo, noopPerm{})
	if registry == nil {
		t.Fatal("registry not created")
	}
	r := gin.New()
	r.Use(gin_mw.ErrorHandler(util.GetStatusCode, ", "))
	group := r.Group("")
	group.Use(AuthMiddleware())
	if _, err := routesAuth.Set(*registry, group); err != nil {
		t.Fatal(err)
	}
	return r
}

func getUsage(r *gin.Engine, target string, userId string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if userId != "" {
		req.Header.Set("X-UserId", userId)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetOperatorUsageById(t *testing.T) {
	repo := &usageRepo{
		PipelineRepository: db.NewMockRepo(),
		usage:              lib.OperatorPipelineUsage{Pipelines: 3, Readable: []lib.PipelineRef{{Id: "p-1", Name: "first"}}},
	}
	w := getUsage(newUsageRouter(t, repo), "/pipeline/statistics/operatorusage/op-1", "user-a")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}
	if want := `{"pipelines":3,"readable":[{"id":"p-1","name":"first"}]}`; w.Body.String() != want {
		t.Errorf("body = %s, want %s", w.Body, want)
	}
	if repo.operatorId != "op-1" || repo.image != "" || repo.user != "user-a" {
		t.Errorf("repository got operator %q, image %q, user %q", repo.operatorId, repo.image, repo.user)
	}
}

func TestGetOperatorUsageByIdPassesImage(t *testing.T) {
	repo := &usageRepo{PipelineRepository: db.NewMockRepo(), usage: lib.OperatorPipelineUsage{Readable: []lib.PipelineRef{}}}
	w := getUsage(newUsageRouter(t, repo), "/pipeline/statistics/operatorusage/op-1?image=img-1", "user-a")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body)
	}
	if want := `{"pipelines":0,"readable":[]}`; w.Body.String() != want {
		t.Errorf("body = %s, want %s", w.Body, want)
	}
	if repo.operatorId != "op-1" || repo.image != "img-1" {
		t.Errorf("repository got operator %q, image %q", repo.operatorId, repo.image)
	}
}

func TestGetOperatorUsageByIdRejectsEmptyImage(t *testing.T) {
	repo := &usageRepo{PipelineRepository: db.NewMockRepo()}
	w := getUsage(newUsageRouter(t, repo), "/pipeline/statistics/operatorusage/op-1?image=", "user-a")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if repo.calls != 0 {
		t.Error("repository queried for an empty image, which would match every version")
	}
}

func TestGetOperatorUsageByIdNeedsUser(t *testing.T) {
	repo := &usageRepo{PipelineRepository: db.NewMockRepo()}
	w := getUsage(newUsageRouter(t, repo), "/pipeline/statistics/operatorusage/op-1", "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if repo.calls != 0 {
		t.Error("repository queried without a user")
	}
}

func TestGetOperatorUsageByIdRepositoryError(t *testing.T) {
	repo := &usageRepo{PipelineRepository: db.NewMockRepo(), err: errors.New("connection refused")}
	w := getUsage(newUsageRouter(t, repo), "/pipeline/statistics/operatorusage/op-1", "user-a")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	if w.Body.String() != MessageSomethingWrong {
		t.Errorf("body = %q, want %q without the cause", w.Body, MessageSomethingWrong)
	}
}
