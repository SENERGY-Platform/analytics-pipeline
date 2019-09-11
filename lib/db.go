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
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Client
var CTX mongo.SessionContext

func InitDB() {
	CTX, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(CTX, options.Client().ApplyURI("mongodb://"+GetEnv("MONGO", "localhost")+":27017"))
	if err != nil {
		panic("failed to connect database: " + err.Error())
	} else {
		fmt.Println("Connected to DB.")
	}
	DB = client
}

func Mongo() *mongo.Collection {
	return DB.Database("service").Collection("pipelines")
}

func CloseDB() {
	err := DB.Disconnect(CTX)
	if err != nil {
		panic("failed to disconnect database: " + err.Error())
	}
}

func GetDB() *mongo.Client {
	return DB
}
