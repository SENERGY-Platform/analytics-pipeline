package lib

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/globalsign/mgo/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PipelineRepository interface {
	InsertPipeline(pipeline Pipeline) (err error)
	UpdatePipeline(pipeline Pipeline, userId string) (err error)
	All(userId string, admin bool, args map[string][]string) (pipelines PipelinesResponse, err error)
	FindPipeline(id string, userId string) (pipeline Pipeline, err error)
	DeletePipeline(id string, userId string, admin bool) (err error)
}

type MongoRepo struct {
}

func NewMongoRepo() *MongoRepo {
	return &MongoRepo{}
}

func (r *MongoRepo) InsertPipeline(pipeline Pipeline) (err error) {
	_, err = Mongo().InsertOne(CTX, pipeline)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (r *MongoRepo) UpdatePipeline(pipeline Pipeline, userId string) (err error) {
	_, err = Mongo().ReplaceOne(CTX, bson.M{"id": pipeline.Id, "userid": userId}, pipeline)

	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (r *MongoRepo) All(userId string, admin bool, args map[string][]string) (pipelines PipelinesResponse, err error) {
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
	pipelines.Data = make([]Pipeline, 0)
	for cur.Next(CTX) {
		// create a value into which the single document can be decoded
		var elem Pipeline
		err = cur.Decode(&elem)
		if err != nil {
			return
		}
		pipelines.Data = append(pipelines.Data, elem)
	}
	return
}

func (r *MongoRepo) FindPipeline(id string, userId string) (pipeline Pipeline, err error) {
	err = Mongo().FindOne(CTX, bson.M{"id": id, "userid": userId}).Decode(&pipeline)
	if err != nil {
		return Pipeline{}, err
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

type MockRepo struct {
}

func NewMockRepo() *MockRepo {
	return &MockRepo{}
}

func (r *MockRepo) InsertPipeline(Pipeline) (err error) {
	return
}

func (r *MockRepo) UpdatePipeline(Pipeline, string) (err error) {
	return
}

func (r *MockRepo) All(string, bool, map[string][]string) (pipelines PipelinesResponse, err error) {
	return
}

func (r *MockRepo) FindPipeline(string, string) (pipeline Pipeline, err error) {
	return
}

func (r *MockRepo) DeletePipeline(string, string, bool) (err error) {
	return
}
