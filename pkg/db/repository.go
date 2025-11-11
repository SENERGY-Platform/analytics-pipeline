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
	All(userId string, admin bool, args map[string][]string) (pipelines lib.PipelinesResponse, err error)
	FindPipeline(id string, userId string) (pipeline lib.Pipeline, err error)
	DeletePipeline(id string, userId string, admin bool) (err error)
	Statistics(userId string, admin bool, args map[string][]string) (statistics lib.PipelineStatistics, err error)
}

type MongoRepo struct {
}

func NewMongoRepo() *MongoRepo {
	return &MongoRepo{}
}

func (r *MongoRepo) InsertPipeline(pipeline lib.Pipeline) (err error) {
	_, err = Mongo().InsertOne(CTX, pipeline)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (r *MongoRepo) UpdatePipeline(pipeline lib.Pipeline, userId string) (err error) {
	_, err = Mongo().ReplaceOne(CTX, bson.M{"id": pipeline.Id, "userid": userId}, pipeline)

	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (r *MongoRepo) All(userId string, admin bool, args map[string][]string) (pipelines lib.PipelinesResponse, err error) {
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
	var cur *mongo.Cursor
	req := bson.M{"userid": userId}
	if val, ok := args["search"]; ok {
		req = bson.M{"userid": userId, "name": bson.M{"$regex": val[0]}}
	}
	if admin {
		req = bson.M{}
	}
	cur, err = Mongo().Find(CTX, req, opt)
	if err != nil {
		return
	}
	req = bson.M{"userid": userId}
	if admin {
		req = bson.M{}
	}
	pipelines.Total, err = Mongo().CountDocuments(CTX, req)
	if err != nil {
		return
	}
	pipelines.Data = make([]lib.Pipeline, 0)
	for cur.Next(CTX) {
		// create a value into which the single document can be decoded
		var elem lib.Pipeline
		err = cur.Decode(&elem)
		if err != nil {
			return
		}
		pipelines.Data = append(pipelines.Data, elem)
	}
	return
}

func (r *MongoRepo) FindPipeline(id string, userId string) (pipeline lib.Pipeline, err error) {
	err = Mongo().FindOne(CTX, bson.M{"id": id, "userid": userId}).Decode(&pipeline)
	if err != nil {
		return lib.Pipeline{}, err
	}
	return pipeline, err
}

func (r *MongoRepo) DeletePipeline(id string, userId string, admin bool) (err error) {
	req := bson.M{"id": id, "userid": userId}
	if admin {
		req = bson.M{"id": id}
	}
	res := Mongo().FindOneAndDelete(CTX, req)
	return res.Err()
}

func (r *MongoRepo) Statistics(_ string, _ bool, _ map[string][]string) (statistics lib.PipelineStatistics, err error) {
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

	if err = aggregate.All(CTX, &statistics.PipelineUserCount); err != nil {
		return
	}

	pipeline = mongo.Pipeline{
		{{"$unwind", "$operators"}},

		{{"$group", bson.D{
			{"_id", "$operators.id"},
			{"count", bson.D{{"$sum", 1}}},
			{"pipelineIds", bson.D{{"$addToSet", "$id"}}},
		}}},

		{{"$sort", bson.D{{"count", -1}}}},
	}

	aggregate, err = Mongo().Aggregate(CTX, pipeline)
	if err != nil {
		return
	}
	defer func(aggregate *mongo.Cursor, ctx context.Context) {
		err = aggregate.Close(ctx)
		if err != nil {
			return
		}
	}(aggregate, CTX)

	if err = aggregate.All(CTX, &statistics.OperatorUsage); err != nil {
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

func (r *MockRepo) All(_ string, _ bool, _ map[string][]string) (pipelines lib.PipelinesResponse, err error) {
	return
}

func (r *MockRepo) FindPipeline(_ string, _ string) (pipeline lib.Pipeline, err error) {
	return
}

func (r *MockRepo) DeletePipeline(_ string, _ string, _ bool) (err error) {
	return
}
