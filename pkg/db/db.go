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
	"strconv"
	"time"

	"github.com/SENERGY-Platform/analytics-pipeline/pkg/config"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Client
var CTX mongo.SessionContext

func InitDB(cfg *config.MongoConfig) {
	CTX, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(CTX, options.Client().ApplyURI("mongodb://"+cfg.Host+":"+strconv.FormatInt(int64(cfg.Port), 10)))
	if err != nil {
		util.Logger.Error("failed to connect database", "error", err)
	} else {
		util.Logger.Info("connected to db")
	}
	DB = client
}

func Mongo() *mongo.Collection {
	return DB.Database("service").Collection("pipelines")
}

func CloseDB() {
	err := DB.Disconnect(CTX)
	if err != nil {
		util.Logger.Error("failed to disconnect database", "error", err)
	}
}
