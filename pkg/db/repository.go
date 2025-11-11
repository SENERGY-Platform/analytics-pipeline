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
	"slices"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PipelineRepository interface {
	InsertPipeline(pipeline lib.Pipeline) (err error)
	UpdatePipeline(pipeline lib.Pipeline, userId string) (err error)
	All(userId string, admin bool, args map[string][]string, ids []string) (pipelines lib.PipelinesResponse, err error)
	FindPipeline(id string, userId string) (pipeline lib.Pipeline, err error)
	DeletePipeline(id string, userId string, admin bool) (err error)
	PipelineUserCount(userId string, admin bool, args map[string][]string) (statistics []lib.PipelineUserCount, err error)
	OperatorUsage(userId string, admin bool, args map[string][]string) (statistics []lib.OperatorUsage, err error)
}

type MongoRepo struct {
}

func NewMongoRepo() *MongoRepo {
	return &MongoRepo{}
}

func (r *MongoRepo) InsertPipeline(pipeline lib.Pipeline) (err error) {
	_, err = Mongo().InsertOne(CTX, pipeline)
	if err != nil {
		return
	}
	return
}

func (r *MongoRepo) UpdatePipeline(pipeline lib.Pipeline, _ string) (err error) {
	_, err = Mongo().ReplaceOne(CTX, bson.M{"id": pipeline.Id}, pipeline)

	if err != nil {
		return err
	}
	return nil
}

func (r *MongoRepo) All(userId string, admin bool, args map[string][]string, ids []string) (pipelines lib.PipelinesResponse, err error) {
	opt := options.Find()
	for arg, value := range args {
		if arg == "limit" {
			limit, _ := strconv.ParseInt(value[0], 10, 64)
			opt.SetLimit(limit)
		}
		if arg == "offset" {
			skip, _ := strconv.ParseInt(value[0], 10, 64)
			opt.SetSkip(skip)
		}
		if arg == "order" {
			ord := strings.Split(value[0], ":")
			order := 1
			if ord[1] == "desc" {
				order = -1
			}
			sortFields := []string{"name", "id", "createdat", "updatedat"}
			if slices.Contains(sortFields, ord[0]) {
				opt.SetSort(bson.M{ord[0]: int64(order)})
			}
		}
	}

	if ids == nil {
		ids = []string{}
	}
	var cur *mongo.Cursor
	req := bson.M{
		"$or": []interface{}{
			bson.M{"id": bson.M{"$in": ids}},
			bson.M{"userid": userId},
		}}
	if val, ok := args["search"]; ok {
		req = bson.M{
			"name": bson.M{
				"$regex":   val[0],
				"$options": "i",
			},
			"$or": []interface{}{
				bson.M{"id": bson.M{"$in": ids}},
				bson.M{"userId": userId},
			}}
	}
	if admin {
		req = bson.M{}
		if val, ok := args["search"]; ok {
			req = bson.M{
				"name": bson.M{
					"$regex":   val[0],
					"$options": "i",
				},
			}
		}
	}
	cur, err = Mongo().Find(CTX, req, opt)
	if err != nil {
		return
	}
	pipelines.Total, err = Mongo().CountDocuments(CTX, req)
	if err != nil {
		return
	}
	pipelines.Data = make([]lib.Pipeline, 0)
	err = cur.All(context.TODO(), &pipelines.Data)
	return
}

func (r *MongoRepo) FindPipeline(id string, _ string) (pipeline lib.Pipeline, err error) {
	err = Mongo().FindOne(CTX, bson.M{"id": id}).Decode(&pipeline)
	return
}

func (r *MongoRepo) DeletePipeline(id string, userId string, admin bool) (err error) {
	req := bson.M{"id": id}
	res := Mongo().FindOneAndDelete(CTX, req)
	return res.Err()
}

func (r *MongoRepo) PipelineUserCount(_ string, _ bool, _ map[string][]string) (statistics []lib.PipelineUserCount, err error) {
	pipeline := mongo.Pipeline{
		{
			{"$group", bson.D{
				{"_id", "$userid"},
				{"count", bson.D{{"$sum", 1}}},
			}},
		},
		{
			{"$sort", bson.D{
				{"count", -1},
			}},
		},
	}

	aggregate, err := Mongo().Aggregate(CTX, pipeline)
	if err != nil {
		return
	}
	defer func(aggregate *mongo.Cursor, ctx context.Context) {
		err = aggregate.Close(ctx)
		if err != nil {
			return
		}
	}(aggregate, CTX)

	if err = aggregate.All(CTX, &statistics); err != nil {
		return
	}
	return
}

func (r *MongoRepo) OperatorUsage(_ string, _ bool, _ map[string][]string) (statistics []lib.OperatorUsage, err error) {
	pipeline := mongo.Pipeline{
		{{"$unwind", "$operators"}},

		{{"$group", bson.D{
			{"_id", "$operators.id"},
			{"count", bson.D{{"$sum", 1}}},
			{"pipelineIds", bson.D{{"$addToSet", "$id"}}},
		}}},

		{{"$sort", bson.D{{"count", -1}}}},
	}

	aggregate, err := Mongo().Aggregate(CTX, pipeline)
	if err != nil {
		return
	}
	defer func(aggregate *mongo.Cursor, ctx context.Context) {
		err = aggregate.Close(ctx)
		if err != nil {
			return
		}
	}(aggregate, CTX)

	if err = aggregate.All(CTX, &statistics); err != nil {
		return
	}
	return
}

type MockRepo struct {
}

func NewMockRepo() *MockRepo {
	return &MockRepo{}
}

func (r *MockRepo) InsertPipeline(_ lib.Pipeline) (err error) {
	return
}

func (r *MockRepo) UpdatePipeline(_ lib.Pipeline, _ string) (err error) {
	return
}

func (r *MockRepo) All(_ string, _ bool, _ map[string][]string, _ []string) (pipelines lib.PipelinesResponse, err error) {
	return
}

func (r *MockRepo) FindPipeline(_ string, _ string) (pipeline lib.Pipeline, err error) {
	return
}

func (r *MockRepo) DeletePipeline(_ string, _ string, _ bool) (err error) {
	return
}

func (r *MockRepo) PipelineUserCount(userId string, admin bool, args map[string][]string) (statistics []lib.PipelineUserCount, err error) {
	return
}
func (r *MockRepo) OperatorUsage(userId string, admin bool, args map[string][]string) (statistics []lib.OperatorUsage, err error) {
	return
}
