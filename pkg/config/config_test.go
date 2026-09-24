/*
 * Copyright 2026 InfAI (CC SES)
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

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	sb_util "github.com/SENERGY-Platform/go-service-base/util"
)

var mongoEnv = []string{"MONGO_CONFIG", "MONGO_URL", "MONGO_USER", "MONGO_PASSWORD", "MONGO_AUTH_SOURCE", "MONGO_DATABASE", "MONGO", "MONGO_PORT"}

// clearMongoEnv unsets the variables for this test; t.Setenv restores them afterwards.
func clearMongoEnv(t *testing.T) {
	for _, k := range mongoEnv {
		t.Setenv(k, "")
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNew_MongoDefaults(t *testing.T) {
	clearMongoEnv(t)
	cfg, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	want := MongoConfig{Url: "mongodb://localhost:27017", AuthSource: "admin", Database: "analytics_pipeline"}
	if cfg.Mongo != want {
		t.Errorf("mongo = %+v, want %+v", cfg.Mongo, want)
	}
}

func TestNew_MongoEnvNames(t *testing.T) {
	clearMongoEnv(t)
	t.Setenv("MONGO_URL", "mongodb://mongo-0:27017/?replicaSet=rs0")
	t.Setenv("MONGO_USER", "analytics-pipeline")
	t.Setenv("MONGO_PASSWORD", "s3cr3t")
	t.Setenv("MONGO_AUTH_SOURCE", "users")
	t.Setenv("MONGO_DATABASE", "other_db")
	cfg, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	want := MongoConfig{
		Url:        "mongodb://mongo-0:27017/?replicaSet=rs0",
		User:       "analytics-pipeline",
		Password:   "s3cr3t",
		AuthSource: "users",
		Database:   "other_db",
	}
	if cfg.Mongo != want {
		t.Errorf("mongo = %+v, want %+v", cfg.Mongo, want)
	}
}

// The loader overwrites a default with "" when the variable is set but empty, which
// is why db rejects an empty database name instead of relying on the default.
func TestNew_EmptyMongoDatabaseEnvOverridesDefault(t *testing.T) {
	clearMongoEnv(t)
	t.Setenv("MONGO_DATABASE", "")
	cfg, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mongo.Database != "" {
		t.Errorf("database = %q, want empty", cfg.Mongo.Database)
	}
}

func TestNew_OldMongoEnvIgnored(t *testing.T) {
	clearMongoEnv(t)
	t.Setenv("MONGO", "old-host")
	t.Setenv("MONGO_PORT", "1234")
	cfg, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mongo.Url != "mongodb://localhost:27017" {
		t.Errorf("url = %q, want the default", cfg.Mongo.Url)
	}
}

func TestNew_MongoConfigFile(t *testing.T) {
	clearMongoEnv(t)
	p := filepath.Join(t.TempDir(), "config.json")
	content := `{"mongo": {"url": "mongodb://file:27017", "user": "u", "password": "s3cr3t", "auth_source": "a", "database": "d"}}`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	want := MongoConfig{Url: "mongodb://file:27017", User: "u", Password: "s3cr3t", AuthSource: "a", Database: "d"}
	if cfg.Mongo != want {
		t.Errorf("mongo = %+v, want %+v", cfg.Mongo, want)
	}
}

// main logs the whole config as JSON at startup.
func TestConfigJSONMasksMongoPassword(t *testing.T) {
	clearMongoEnv(t)
	t.Setenv("MONGO_USER", "analytics-pipeline")
	t.Setenv("MONGO_PASSWORD", "s3cr3t")
	cfg, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	if s := sb_util.ToJsonStr(cfg); strings.Contains(s, "s3cr3t") {
		t.Errorf("config JSON leaks the password: %s", s)
	}
}
