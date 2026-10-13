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
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Every method takes a context. It carries the deadline of the request that caused
// the query and the trace it belongs to, which is what makes otelmongo's spans
// children of that request rather than roots of their own.
type PipelineRepository interface {
	InsertPipeline(ctx context.Context, pipeline lib.Pipeline) (err error)
	UpdatePipeline(ctx context.Context, pipeline lib.Pipeline, userId string) (err error)
	All(ctx context.Context, userId string, admin bool, args map[string][]string, ids []string) (pipelines lib.PipelinesResponse, err error)
	FindPipeline(ctx context.Context, id string, userId string) (pipeline lib.Pipeline, err error)
	DeletePipeline(ctx context.Context, id string, userId string, admin bool) (err error)
	PipelineUserCount(ctx context.Context, userId string, admin bool, args map[string][]string) (statistics []lib.PipelineUserCount, err error)
	OperatorUsage(ctx context.Context, userId string, admin bool, args map[string][]string) (statistics []lib.OperatorUsage, err error)
	FlowUsage(ctx context.Context, id string) (statistics []lib.FlowUsage, err error)
	// OperatorPipelineUsage matches imageId only when it is not empty.
	OperatorPipelineUsage(ctx context.Context, operatorId string, imageId string, userId string) (usage lib.OperatorPipelineUsage, err error)
}

// MongoRepo drops ts_conn from operator configs on every write and every read, see withoutTsConn.
type MongoRepo struct {
}

func NewMongoRepo() *MongoRepo {
	return &MongoRepo{}
}

func (r *MongoRepo) InsertPipeline(ctx context.Context, pipeline lib.Pipeline) (err error) {
	_, err = Mongo().InsertOne(ctx, withoutTsConn(pipeline))
	if err != nil {
		return
	}
	return
}

func (r *MongoRepo) UpdatePipeline(ctx context.Context, pipeline lib.Pipeline, _ string) (err error) {
	_, err = Mongo().ReplaceOne(ctx, bson.M{"id": pipeline.Id}, withoutTsConn(pipeline))

	if err != nil {
		return err
	}
	return nil
}

func (r *MongoRepo) All(ctx context.Context, userId string, admin bool, args map[string][]string, ids []string) (pipelines lib.PipelinesResponse, err error) {

	opt := options.Find()

	for arg, value := range args {

		switch arg {

		case "limit":
			limit, _ := strconv.ParseInt(value[0], 10, 64)
			opt.SetLimit(limit)

		case "offset":
			skip, _ := strconv.ParseInt(value[0], 10, 64)
			opt.SetSkip(skip)

		case "order":
			ord := strings.SplitN(value[0], ":", 2)
			if len(ord) != 2 {
				break
			}

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

	andFilters := bson.A{}

	if !admin {
		if ids == nil {
			ids = []string{}
		}

		andFilters = append(andFilters, bson.M{
			"$or": bson.A{
				bson.M{"id": bson.M{"$in": ids}},
				bson.M{"userid": userId},
			},
		})
	}

	if val, ok := args["search"]; ok && len(val) > 0 {
		andFilters = append(andFilters, bson.M{
			"name": bson.M{
				"$regex":   val[0],
				"$options": "i",
			},
		})
	}

	fieldMap := map[string]string{
		"operator": "operators.operatorid",
		"flow":     "flowid",
	}

	if vals, ok := args["filter"]; ok {
		for _, raw := range vals {
			for _, f := range strings.Split(raw, "|") {

				parts := strings.SplitN(f, ":", 2)
				if len(parts) != 2 {
					continue
				}

				field, exists := fieldMap[parts[0]]
				if !exists {
					continue
				}

				values := strings.Split(parts[1], ",")
				if len(values) == 0 {
					continue
				}

				andFilters = append(andFilters, bson.M{
					field: bson.M{"$in": values},
				})
			}
		}
	}

	req := bson.M{}
	if len(andFilters) > 0 {
		req["$and"] = andFilters
	}

	var cur *mongo.Cursor
	cur, err = Mongo().Find(ctx, req, opt)
	if err != nil {
		return
	}

	pipelines.Total, err = Mongo().CountDocuments(ctx, req)
	if err != nil {
		return
	}

	pipelines.Data = make([]lib.Pipeline, 0)
	err = cur.All(ctx, &pipelines.Data)
	// Documents written before ts_conn was dropped on write may still carry it.
	for i := range pipelines.Data {
		pipelines.Data[i] = withoutTsConn(pipelines.Data[i])
	}
	return
}

func (r *MongoRepo) FindPipeline(ctx context.Context, id string, _ string) (pipeline lib.Pipeline, err error) {
	err = Mongo().FindOne(ctx, bson.M{"id": id}).Decode(&pipeline)
	return withoutTsConn(pipeline), err
}

func (r *MongoRepo) DeletePipeline(ctx context.Context, id string, userId string, admin bool) (err error) {
	req := bson.M{"id": id}
	res := Mongo().FindOneAndDelete(ctx, req)
	return res.Err()
}

func (r *MongoRepo) PipelineUserCount(ctx context.Context, _ string, _ bool, _ map[string][]string) (statistics []lib.PipelineUserCount, err error) {
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

	aggregate, err := Mongo().Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	defer func(aggregate *mongo.Cursor, ctx context.Context) {
		err = aggregate.Close(ctx)
		if err != nil {
			return
		}
	}(aggregate, ctx)

	if err = aggregate.All(ctx, &statistics); err != nil {
		return
	}
	return
}

func (r *MongoRepo) OperatorUsage(ctx context.Context, _ string, _ bool, _ map[string][]string) (statistics []lib.OperatorUsage, err error) {
	pipeline := mongo.Pipeline{
		{{"$unwind", "$operators"}},

		{{"$group", bson.D{
			{"_id", "$operators.operatorid"},
			{"count", bson.D{{"$sum", 1}}},
			{"pipelineIds", bson.D{{"$addToSet", "$id"}}},
		}}},

		{{"$sort", bson.D{{"count", -1}}}},
	}

	aggregate, err := Mongo().Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	defer func(aggregate *mongo.Cursor, ctx context.Context) {
		err = aggregate.Close(ctx)
		if err != nil {
			return
		}
	}(aggregate, ctx)

	if err = aggregate.All(ctx, &statistics); err != nil {
		return
	}
	return
}

func (r *MongoRepo) FlowUsage(ctx context.Context, id string) (statistics []lib.FlowUsage, err error) {
	pipeline := mongo.Pipeline{}

	if id != "" {
		pipeline = append(pipeline, bson.D{
			{"$match", bson.D{
				{"flowid", id},
			}},
		})
	}

	pipeline = append(pipeline,
		bson.D{{"$group", bson.D{
			{"_id", "$flowid"},
			{"count", bson.D{{"$sum", 1}}},
			{"pipelineIds", bson.D{{"$addToSet", "$id"}}},
		}}},
		bson.D{{"$sort", bson.D{{"count", -1}}}},
	)

	aggregate, err := Mongo().Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	defer func(aggregate *mongo.Cursor, ctx context.Context) {
		err = aggregate.Close(ctx)
		if err != nil {
			return
		}
	}(aggregate, ctx)

	if err = aggregate.All(ctx, &statistics); err != nil {
		return
	}
	return
}

// OperatorPipelineUsage runs as one aggregation so that the caller's pipelines are
// taken from the same documents that were counted.
func (r *MongoRepo) OperatorPipelineUsage(ctx context.Context, operatorId string, imageId string, userId string) (usage lib.OperatorPipelineUsage, err error) {
	// Both conditions inside one $elemMatch: the image has to belong to the same
	// operator entry, not to any other operator of the pipeline.
	elem := bson.M{"operatorid": operatorId}
	if imageId != "" {
		elem["imageid"] = imageId
	}
	// An empty caller id must not list the pipelines that have no owner.
	own := bson.M{"userid": userId}
	if userId == "" {
		own = bson.M{"_id": bson.M{"$exists": false}}
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"operators": bson.M{"$elemMatch": elem}}}},
		{{Key: "$facet", Value: bson.M{
			"count": bson.A{bson.M{"$count": "n"}},
			"readable": bson.A{
				bson.M{"$match": own},
				bson.M{"$project": bson.M{"_id": 0, "id": 1, "name": 1}},
			},
		}}},
	}

	aggregate, err := Mongo().Aggregate(ctx, pipeline)
	if err != nil {
		return
	}
	defer func() {
		_ = aggregate.Close(ctx)
	}()

	var result []struct {
		Count []struct {
			N int64 `bson:"n"`
		} `bson:"count"`
		Readable []lib.PipelineRef `bson:"readable"`
	}
	if err = aggregate.All(ctx, &result); err != nil {
		return
	}
	usage.Readable = []lib.PipelineRef{}
	// $facet always returns one document; $count inside it yields nothing for no match.
	if len(result) == 1 {
		if len(result[0].Count) == 1 {
			usage.Pipelines = result[0].Count[0].N
		}
		usage.Readable = append(usage.Readable, result[0].Readable...)
	}
	slices.SortFunc(usage.Readable, func(a, b lib.PipelineRef) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Id, b.Id))
	})
	return
}

type MockRepo struct {
}

func NewMockRepo() *MockRepo {
	return &MockRepo{}
}

func (r *MockRepo) InsertPipeline(_ context.Context, _ lib.Pipeline) (err error) {
	return
}

func (r *MockRepo) UpdatePipeline(_ context.Context, _ lib.Pipeline, _ string) (err error) {
	return
}

func (r *MockRepo) All(_ context.Context, _ string, _ bool, _ map[string][]string, _ []string) (pipelines lib.PipelinesResponse, err error) {
	return
}

func (r *MockRepo) FindPipeline(_ context.Context, _ string, _ string) (pipeline lib.Pipeline, err error) {
	return
}

func (r *MockRepo) DeletePipeline(_ context.Context, _ string, _ string, _ bool) (err error) {
	return
}

func (r *MockRepo) PipelineUserCount(_ context.Context, _ string, _ bool, _ map[string][]string) (statistics []lib.PipelineUserCount, err error) {
	return
}
func (r *MockRepo) OperatorUsage(_ context.Context, _ string, _ bool, _ map[string][]string) (statistics []lib.OperatorUsage, err error) {
	return
}

func (r *MockRepo) FlowUsage(_ context.Context, id string) (statistics []lib.FlowUsage, err error) {
	return
}

func (r *MockRepo) OperatorPipelineUsage(_ context.Context, _ string, _ string, _ string) (usage lib.OperatorPipelineUsage, err error) {
	return lib.OperatorPipelineUsage{Readable: []lib.PipelineRef{}}, nil
}
