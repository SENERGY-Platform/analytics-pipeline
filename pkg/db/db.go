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

package db

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/SENERGY-Platform/analytics-pipeline/pkg/config"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	"github.com/SENERGY-Platform/gin-middleware/otelx"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

var DB *mongo.Client

// InitDB connects to mongo and instruments the connection.
//
// otelmongo's monitor turns every command the driver sends into a span. It needs a
// tracer provider to exist, which is what the otelx call sets up — the same
// initialisation the HTTP middleware performs, and it happens only once, whichever
// call gets there first.
//
// The spans are children of whatever the caller's context carries, so a query shows
// up under the request that caused it. That only works because the context now
// travels from the handler down to the repository: it used to be a package variable
// that this function shadowed with ':=', so it stayed nil forever, every call site
// passed nil, and the driver quietly substituted context.Background(). Queries had
// no deadline and no trace to belong to.
func InitDB(ctx context.Context, cfg *config.MongoConfig, serviceName, otelEndpoint string) error {
	// Returned rather than logged. otelx initialises behind a sync.Once, so this is
	// the only call that can fail: every later one — the HTTP middleware's — gets a
	// nil error back without anything having happened, and would build a handler
	// around the global no-op propagator. Nothing would be traced and nothing would
	// say so.
	if _, err := otelx.GinOpenTelemetry(ctx, serviceName, otelEndpoint); err != nil {
		return fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().
		ApplyURI("mongodb://"+cfg.Host+":"+strconv.FormatInt(int64(cfg.Port), 10)).
		SetMonitor(otelmongo.NewMonitor()))
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	util.Logger.InfoContext(ctx, "connected to db")
	DB = client
	return nil
}

func Mongo() *mongo.Collection {
	return DB.Database("service").Collection("pipelines")
}

func CloseDB() {
	// Its own context: this runs while the process shuts down, so the one the service
	// ran under is already cancelled and a disconnect using it would fail at once.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := DB.Disconnect(ctx); err != nil {
		util.Logger.ErrorContext(ctx, "failed to disconnect database", "error", err)
	}
}
