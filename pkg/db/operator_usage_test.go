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

package db

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// useTestMongo points the package at a scratch database on MONGO_TEST_URL and skips without it.
func useTestMongo(t *testing.T) {
	t.Helper()
	url := os.Getenv("MONGO_TEST_URL")
	if testing.Short() || url == "" {
		t.Skip("needs MONGO_TEST_URL; not in -short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(url))
	if err != nil {
		t.Fatal(err)
	}
	if err = client.Ping(ctx, nil); err != nil {
		t.Fatal(err)
	}
	oldDB, oldDatabase := DB, database
	DB, database = client, "pipeline_usage_test_"+primitive.NewObjectID().Hex()
	t.Cleanup(func() {
		_ = client.Database(database).Drop(context.Background())
		_ = client.Disconnect(context.Background())
		DB, database = oldDB, oldDatabase
	})
}

type op struct{ operatorId, imageId string }

// insertPipeline stores the pipeline through lib.Pipeline so the test sees the bson
// field names production writes.
func insertPipeline(t *testing.T, id, name, userId string, ops ...op) {
	t.Helper()
	p := lib.Pipeline{Id: id, Name: name, UserId: userId}
	for _, o := range ops {
		p.Operators = append(p.Operators, lib.Operator{Id: primitive.NewObjectID().Hex(), OperatorId: o.operatorId, ImageId: o.imageId})
	}
	if _, err := Mongo().InsertOne(context.Background(), p); err != nil {
		t.Fatal(err)
	}
}

func TestMongoOperatorPipelineUsage(t *testing.T) {
	useTestMongo(t)
	// Inserted out of name and id order, so the sort is the repository's and not the insert order.
	insertPipeline(t, "p-2", "beta", "user-a", op{"op-1", "img-1"})
	insertPipeline(t, "p-9", "alpha", "user-a", op{"op-1", "img-1"}, op{"op-1", "img-1"})
	insertPipeline(t, "p-1", "beta", "user-a", op{"op-1", "img-2"}, op{"op-2", "img-1"})
	insertPipeline(t, "p-3", "gamma", "user-b", op{"op-1", "img-1"})
	insertPipeline(t, "p-4", "ownerless", "", op{"op-1", "img-1"})
	insertPipeline(t, "p-5", "other operator", "user-a", op{"op-2", "img-2"})
	insertPipeline(t, "p-6", "no operators", "user-a")

	ref := func(id, name string) lib.PipelineRef { return lib.PipelineRef{Id: id, Name: name} }
	cases := []struct {
		name                      string
		operatorId, imageId, user string
		wantPipelines             int64
		wantReadable              []lib.PipelineRef
	}{
		{"all users, operator twice counts once", "op-1", "", "user-a", 5,
			[]lib.PipelineRef{ref("p-9", "alpha"), ref("p-1", "beta"), ref("p-2", "beta")}},
		{"image narrows, other entry's image does not match", "op-1", "img-1", "user-a", 4,
			[]lib.PipelineRef{ref("p-9", "alpha"), ref("p-2", "beta")}},
		{"other entry's image does not match the other way round", "op-2", "img-2", "user-a", 1,
			[]lib.PipelineRef{ref("p-5", "other operator")}},
		{"same entry matches", "op-2", "img-1", "user-a", 1,
			[]lib.PipelineRef{ref("p-1", "beta")}},
		{"readable holds only the caller's", "op-1", "img-2", "user-b", 1, []lib.PipelineRef{}},
		{"empty caller lists no ownerless pipeline", "op-1", "", "", 5, []lib.PipelineRef{}},
		{"no match", "op-3", "", "user-a", 0, []lib.PipelineRef{}},
		{"no match for an unknown image", "op-1", "img-3", "user-a", 0, []lib.PipelineRef{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			usage, err := (&MongoRepo{}).OperatorPipelineUsage(context.Background(), c.operatorId, c.imageId, c.user)
			if err != nil {
				t.Fatal(err)
			}
			if usage.Pipelines != c.wantPipelines {
				t.Errorf("pipelines = %d, want %d", usage.Pipelines, c.wantPipelines)
			}
			if !slices.Equal(usage.Readable, c.wantReadable) {
				t.Errorf("readable = %v, want %v", usage.Readable, c.wantReadable)
			}
			if usage.Readable == nil {
				t.Error("readable is nil and would encode as null")
			}
		})
	}
}

func TestMongoOperatorPipelineUsageEncodesNoMatchAsEmptyList(t *testing.T) {
	useTestMongo(t)
	usage, err := (&MongoRepo{}).OperatorPipelineUsage(context.Background(), "op-1", "", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(usage)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"pipelines":0,"readable":[]}`; string(b) != want {
		t.Errorf("body = %s, want %s", b, want)
	}
}
