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
	"errors"
	"fmt"
	"time"

	"github.com/SENERGY-Platform/analytics-pipeline/pkg/config"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	"github.com/SENERGY-Platform/gin-middleware/otelx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

var DB *mongo.Client

// database is the name InitDB was configured with; Mongo() reads it.
var database string

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

	opts, err := clientOptions(cfg)
	if err != nil {
		return err
	}
	client, err := connect(ctx, opts.SetMonitor(otelmongo.NewMonitor()), cfg.Database, 10*time.Second)
	if err != nil {
		return err
	}
	util.Logger.InfoContext(ctx, "connected to db", "database", cfg.Database)
	DB = client
	database = cfg.Database
	return nil
}

// connect runs an authenticated command on the service's database because
// mongo.Connect is lazy and ping needs no auth: otherwise an unreachable server or
// wrong or missing credentials would only show up at the first request.
func connect(ctx context.Context, opts *options.ClientOptions, database string, timeout time.Duration) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}
	_, err = client.Database(database).ListCollectionNames(ctx, bson.D{},
		options.ListCollections().SetNameOnly(true).SetAuthorizedCollections(true))
	if err != nil {
		disconnectCtx, cancelDisconnect := context.WithTimeout(context.Background(), timeout)
		defer cancelDisconnect()
		_ = client.Disconnect(disconnectCtx)
		return nil, fmt.Errorf("mongo startup check failed: %w", err)
	}
	return client, nil
}

// clientOptions turns the config into driver options without touching the network.
func clientOptions(cfg *config.MongoConfig) (*options.ClientOptions, error) {
	if cfg.Database == "" {
		return nil, errors.New("mongo database name must not be empty")
	}
	opts := options.Client().ApplyURI(cfg.Url)
	if cfg.User != "" {
		// SCRAM, the only mechanism reachable here, cannot authenticate without a
		// password; failing now beats every query failing later.
		if cfg.Password.Value() == "" {
			return nil, errors.New("mongo user is set but password is empty")
		}
		opts.SetAuth(options.Credential{
			Username:   cfg.User,
			Password:   cfg.Password.Value(),
			AuthSource: cfg.AuthSource,
		})
	}
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid mongo client options: %w", err)
	}
	return opts, nil
}

func Mongo() *mongo.Collection {
	return DB.Database(database).Collection("pipelines")
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
