package lib

import (
	"fmt"
	"log"
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
			opt.SetSort(bson.M{ord[0]: int64(order)})
		}
	}
	var cur *mongo.Cursor
	req := bson.M{"userid": userId}
	if val, ok := args["search"]; ok {
		req = bson.M{"userid": userId, "_id": bson.RegEx{Pattern: val[0], Options: "i"}}
	}
	if admin {
		req = bson.M{}
	}
	cur, err = Mongo().Find(CTX, req, opt)
	if err != nil {
		log.Println(err)
		return
	}

	for cur.Next(CTX) {
		// create a value into which the single document can be decoded
		var elem Pipeline
		err = cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
			return
		}
		pipelines.Data = append(pipelines.Data, elem)
	}
	pipelines.Total = len(pipelines.Data)
	return
}

func (r *MongoRepo) FindPipeline(id string, userId string) (pipeline Pipeline, err error) {
	err = Mongo().FindOne(CTX, bson.M{"id": id, "userid": userId}).Decode(&pipeline)
	if err != nil {
		log.Println(err)
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

func (r *MockRepo) InsertPipeline(pipeline Pipeline) (err error) {
	return
}

func (r *MockRepo) UpdatePipeline(pipeline Pipeline, userId string) (err error) {
	return
}

func (r *MockRepo) All(userId string, admin bool, args map[string][]string) (pipelines PipelinesResponse, err error) {
	return
}

func (r *MockRepo) FindPipeline(id string, userId string) (pipeline Pipeline, err error) {
	return
}

func (r *MockRepo) DeletePipeline(id string, userId string, admin bool) (err error) {
	return
}
