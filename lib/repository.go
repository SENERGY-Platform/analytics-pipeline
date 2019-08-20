package lib

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/globalsign/mgo/bson"
)

type PipelineRepository interface {
	InsertPipeline(pipeline Pipeline)
	All(userId string, args map[string][]string) (pipelines []Pipeline)
	FindPipeline(id string, userId string) (pipeline Pipeline)
	DeletePipeline(id string, userId string) (err error)
}

type MongoRepo struct {
}

func NewMongoRepo() *MongoRepo {
	return &MongoRepo{}
}

func (r *MongoRepo) InsertPipeline(pipeline Pipeline) {
	err := Mongo().Insert(pipeline)
	if err != nil {
		fmt.Println(err)
	}
}

func (r *MongoRepo) All(userId string, args map[string][]string) (pipelines []Pipeline) {
	tx := Mongo().Find(bson.M{"userid": userId})
	if val, ok := args["search"]; ok {
		tx = Mongo().Find(bson.M{"userid": userId, "_id": bson.RegEx{Pattern: val[0], Options: "i"}})
	}
	for arg, value := range args {
		if arg == "limit" {
			limit, _ := strconv.Atoi(value[0])
			tx = tx.Limit(limit)
		}
		if arg == "offset" {
			skip, _ := strconv.Atoi(value[0])
			tx = tx.Limit(skip)
		}
		if arg == "order" {
			ord := strings.Split(value[0], ":")
			order := ord[0]
			if ord[1] == "desc" {
				order = "-" + ord[0]
			}
			tx = tx.Sort(order)
		}
	}
	err := tx.All(&pipelines)
	if err != nil {
		fmt.Println(err)
	}
	return
}

func (r *MongoRepo) FindPipeline(id string, userId string) (pipeline Pipeline) {
	err := Mongo().Find(bson.M{"id": id, "userid": userId}).One(&pipeline)
	if err != nil {
		fmt.Println(err)
	}
	return
}

func (r *MongoRepo) DeletePipeline(id string, userId string) (err error) {
	var pipeline Pipeline
	err = Mongo().Find(bson.M{"id": id, "userid": userId}).One(&pipeline)
	if err != nil {
		return
	}
	err = Mongo().Remove(&pipeline)
	if err != nil {
		return
	}
	return
}

type MockRepo struct {
}

func NewMockRepo() *MockRepo {
	return &MockRepo{}
}

func (r *MockRepo) InsertPipeline(pipeline Pipeline) {

}

func (r *MockRepo) All(userId string, args map[string][]string) (pipelines []Pipeline) {
	return
}

func (r *MockRepo) FindPipeline(id string, userId string) (pipeline Pipeline) {
	return
}

func (r *MockRepo) DeletePipeline(id string, userId string) (err error) {
	return
}
