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

package db

import (
	"context"
	"net"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/SENERGY-Platform/analytics-pipeline/pkg/config"
	"github.com/SENERGY-Platform/go-service-base/config-hdl/types"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestClientOptions_AuthWhenUserGiven(t *testing.T) {
	opts, err := clientOptions(&config.MongoConfig{
		Url:        "mongodb://localhost:27017",
		User:       "analytics-pipeline",
		Password:   "s3cr3t",
		AuthSource: "admin",
		Database:   "analytics_pipeline",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := &options.Credential{Username: "analytics-pipeline", Password: "s3cr3t", AuthSource: "admin"}
	if !reflect.DeepEqual(opts.Auth, want) {
		t.Errorf("auth = %+v, want %+v", opts.Auth, want)
	}
}

func TestClientOptions_NoAuthWhenUserEmpty(t *testing.T) {
	// A password without a user must not switch auth on.
	opts, err := clientOptions(&config.MongoConfig{
		Url:        "mongodb://localhost:27017",
		Password:   "s3cr3t",
		AuthSource: "admin",
		Database:   "analytics_pipeline",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Auth != nil {
		t.Errorf("auth = %+v, want nil", opts.Auth)
	}
}

func TestClientOptions_ExplicitUserReplacesURICredentials(t *testing.T) {
	opts, err := clientOptions(&config.MongoConfig{
		Url:        "mongodb://old:oldpw@localhost:27017/?authSource=other&authMechanism=SCRAM-SHA-1",
		User:       "new",
		Password:   "newpw",
		AuthSource: "admin",
		Database:   "analytics_pipeline",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := &options.Credential{Username: "new", Password: "newpw", AuthSource: "admin"}
	if !reflect.DeepEqual(opts.Auth, want) {
		t.Errorf("auth = %+v, want %+v", opts.Auth, want)
	}
}

func TestClientOptions_URIPassedUnchanged(t *testing.T) {
	uri := "mongodb://mongo-0.mongo:27017,mongo-1.mongo:27017/?replicaSet=rs0&readPreference=primary"
	opts, err := clientOptions(&config.MongoConfig{Url: uri, Database: "analytics_pipeline"})
	if err != nil {
		t.Fatal(err)
	}
	if got := opts.GetURI(); got != uri {
		t.Errorf("uri = %q, want %q", got, uri)
	}
	if want := []string{"mongo-0.mongo:27017", "mongo-1.mongo:27017"}; !reflect.DeepEqual(opts.Hosts, want) {
		t.Errorf("hosts = %v, want %v", opts.Hosts, want)
	}
	if opts.ReplicaSet == nil || *opts.ReplicaSet != "rs0" {
		t.Errorf("replica set = %v, want rs0", opts.ReplicaSet)
	}
}

func TestClientOptions_Rejects(t *testing.T) {
	cases := map[string]config.MongoConfig{
		// The old MONGO value carried over without a scheme.
		"uri without scheme":    {Url: "localhost:27017", User: "u", Password: "s3cr3t", Database: "analytics_pipeline"},
		"empty uri":             {Url: "", Database: "analytics_pipeline"},
		"user without password": {Url: "mongodb://localhost:27017", User: "u", Database: "analytics_pipeline"},
		"empty database":        {Url: "mongodb://localhost:27017"},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := clientOptions(&cfg)
			if err == nil {
				t.Fatal("expected an error")
			}
			if strings.Contains(err.Error(), "s3cr3t") {
				t.Errorf("error leaks the password: %v", err)
			}
		})
	}
}

func TestConnect_FailsWithoutServer(t *testing.T) {
	// A port that was just free has no server behind it, so only the startup check can fail.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	if err = l.Close(); err != nil {
		t.Fatal(err)
	}
	opts, err := clientOptions(&config.MongoConfig{Url: "mongodb://" + addr + "/?directConnection=true", Database: "analytics_pipeline"})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	client, err := connect(context.Background(), opts, "analytics_pipeline", 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected an error")
	}
	if client != nil {
		t.Error("expected no client on failure")
	}
	if !strings.Contains(err.Error(), "mongo startup check failed") {
		t.Errorf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("connect took %v, the timeout was not applied", elapsed)
	}
}

// TestConnect_Auth needs a server with access control, a readWrite user on
// analytics_pipeline and a readWrite user on some other database, both in admin.
func TestConnect_Auth(t *testing.T) {
	vars := []string{"MONGO_AUTH_TEST_URL", "MONGO_AUTH_TEST_USER", "MONGO_AUTH_TEST_PASSWORD", "MONGO_AUTH_TEST_OTHER_USER", "MONGO_AUTH_TEST_OTHER_PASSWORD"}
	for _, v := range vars {
		if testing.Short() || os.Getenv(v) == "" {
			t.Skip("needs " + strings.Join(vars, ", ") + "; not in -short")
		}
	}
	url, user, password := os.Getenv("MONGO_AUTH_TEST_URL"), os.Getenv("MONGO_AUTH_TEST_USER"), os.Getenv("MONGO_AUTH_TEST_PASSWORD")
	otherUser, otherPassword := os.Getenv("MONGO_AUTH_TEST_OTHER_USER"), os.Getenv("MONGO_AUTH_TEST_OTHER_PASSWORD")
	cases := []struct {
		name     string
		user     string
		password string
		wantErr  bool
	}{
		{"correct credentials", user, password, false},
		{"wrong password", user, password + "-wrong", true},
		{"no credentials", "", "", true},
		{"user of another database", otherUser, otherPassword, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts, err := clientOptions(&config.MongoConfig{Url: url, User: c.user, Password: types.Secret(c.password), AuthSource: "admin", Database: "analytics_pipeline"})
			if err != nil {
				t.Fatal(err)
			}
			client, err := connect(context.Background(), opts, "analytics_pipeline", 10*time.Second)
			if client != nil {
				_ = client.Disconnect(context.Background())
			}
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if err != nil && (strings.Contains(err.Error(), password) || strings.Contains(err.Error(), otherPassword) || !strings.Contains(err.Error(), "mongo startup check failed")) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
