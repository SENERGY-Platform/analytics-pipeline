/*
 * Copyright 2018 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lib

import (
	"fmt"

	"strings"

	"strconv"

	"github.com/globalsign/mgo/bson"
	"github.com/satori/go.uuid"
)

func savePipeline(pipeline Pipeline, user_id string) uuid.UUID {
	fmt.Println("Save Pipeline Data")
	id := uuid.NewV4()
	pipeline.Id = id.String()
	pipeline.UserId = user_id
	fmt.Println(pipeline)
	Mongo().Insert(pipeline)
	return id
}

func getPipelines(user_id string, args map[string][]string) (pipelines []Pipeline) {
	tx := Mongo().Find(bson.M{"userid": user_id})
	if val, ok := args["search"]; ok {
		tx = Mongo().Find(bson.M{"userid": user_id, "_id": bson.RegEx{Pattern: val[0], Options: "i"}})
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
	tx.All(&pipelines)
	return
}

func getPipeline(id string, user_id string) (pipeline Pipeline) {
	Mongo().Find(bson.M{"id": id, "userid": user_id}).One(&pipeline)
	return
}

func deletePipeline(id string, user_id string) Response {
	var pipeline Pipeline
	Mongo().Find(bson.M{"id": id, "userid": user_id}).One(&pipeline)
	Mongo().Remove(&pipeline)
	return Response{"OK"}
}
