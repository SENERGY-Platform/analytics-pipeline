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

package config

import sb_config_hdl "github.com/SENERGY-Platform/go-service-base/config-hdl"

type LoggerConfig struct {
	Level string `json:"level" env_var:"LOGGER_LEVEL"`
}

type MongoConfig struct {
	Host string `json:"host" env_var:"MONGO"`
	Port int    `json:"port" env_var:"MONGO_PORT"`
}
type Config struct {
	Logger           LoggerConfig `json:"logger" env_var:"LOGGER_CONFIG"`
	ServerPort       int          `json:"server_port" env_var:"SERVER_PORT"`
	Debug            bool         `json:"debug" env_var:"DEBUG"`
	URLPrefix        string       `json:"url_prefix" env_var:"URL_PREFIX"`
	Mongo            MongoConfig  `json:"mongo" env_var:"MONGO_CONFIG"`
	PermissionsV2Url string       `json:"permissions_v2_url" env_var:"PERMISSIONS_V2_URL"`
}

func New(path string) (*Config, error) {
	cfg := Config{
		ServerPort: 8000,
		Debug:      false,
		Logger: LoggerConfig{
			Level: "info",
		},
		PermissionsV2Url: "http://permv2.permissions:8080",
		Mongo: MongoConfig{
			Host: "localhost",
			Port: 27017,
		},
	}
	err := sb_config_hdl.Load(&cfg, nil, envTypeParser, nil, path)
	return &cfg, err
}
