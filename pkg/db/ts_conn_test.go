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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// fakeDSN is built at run time so no DSN-shaped literal sits in the source.
func fakeDSN() string {
	return "postgres" + "://" + "operator" + ":" + "pw-" + primitive.NewObjectID().Hex() + "@" + "timescale" + ":5432/db"
}

// captureLog sends util.Logger into a buffer for the duration of the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	old := util.Logger
	util.Logger = slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	t.Cleanup(func() { util.Logger = old })
	return buf
}

// storedConfigs reads the operator configs of pipeline id straight from the collection,
// bypassing the repository's read path.
func storedConfigs(t *testing.T, id string) []map[string]string {
	t.Helper()
	var p lib.Pipeline
	if err := Mongo().FindOne(context.Background(), bson.M{"id": id}).Decode(&p); err != nil {
		t.Fatal(err)
	}
	configs := make([]map[string]string, 0, len(p.Operators))
	for _, o := range p.Operators {
		configs = append(configs, o.Config)
	}
	return configs
}

func checkConfigs(t *testing.T, what string, got []map[string]string, want ...map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %d operators, want %d", what, len(got), len(want))
	}
	for i := range want {
		if _, ok := got[i][tsConnKey]; ok {
			t.Errorf("%s: operator %d still has ts_conn", what, i)
		}
		if !maps.Equal(got[i], want[i]) || (got[i] == nil) != (want[i] == nil) {
			t.Errorf("%s: operator %d config keys %v, want %v", what, i, keyList(got[i]), keyList(want[i]))
		}
	}
}

// keyList names the keys only, so a failing test never prints a DSN.
func keyList(m map[string]string) string {
	if m == nil {
		return "<nil>"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return fmt.Sprint(keys)
}

func TestWithoutTsConn(t *testing.T) {
	dsn := fakeDSN()
	in := lib.Pipeline{Id: "p-1", Operators: []lib.Operator{
		{Id: "o-1", Config: map[string]string{tsConnKey: dsn, "window": "5"}},
		{Id: "o-2", Config: nil},
		{Id: "o-3", Config: map[string]string{"window": "7"}},
		{Id: "o-4", Config: map[string]string{tsConnKey: ""}},
	}}
	out := withoutTsConn(in)

	got := make([]map[string]string, 0, len(out.Operators))
	for _, o := range out.Operators {
		got = append(got, o.Config)
	}
	checkConfigs(t, "result", got,
		map[string]string{"window": "5"}, nil, map[string]string{"window": "7"}, map[string]string{})
	if in.Operators[0].Config[tsConnKey] != dsn || in.Operators[3].Config == nil {
		t.Error("the caller's config map was modified")
	}
	if out.Operators[0].Id != "o-1" || out.Id != "p-1" {
		t.Error("fields other than the config changed")
	}
	if b, _ := json.Marshal(out); strings.Contains(string(b), tsConnKey) {
		t.Error("ts_conn in the JSON encoding")
	}

	if withoutTsConn(lib.Pipeline{}).Operators != nil {
		t.Error("nil operators became non-nil")
	}
	if ops := withoutTsConn(lib.Pipeline{Operators: []lib.Operator{}}).Operators; ops == nil || len(ops) != 0 {
		t.Error("empty operators changed")
	}
}

func TestErrorAttrsWithholdsServerMessage(t *testing.T) {
	dsn := fakeDSN()
	cases := map[string]error{
		"write exception": mongo.WriteException{
			WriteErrors:       mongo.WriteErrors{{Code: 2, Message: "cannot apply to " + dsn}},
			WriteConcernError: &mongo.WriteConcernError{Code: 64, Message: dsn},
		},
		"command error":         fmt.Errorf("wrapped: %w", mongo.CommandError{Code: 2, Name: "BadValue", Message: dsn}),
		"other server error":    mongo.BulkWriteException{WriteErrors: []mongo.BulkWriteError{{WriteError: mongo.WriteError{Message: dsn}}}},
		"wrapped write failure": fmt.Errorf("update: %w", mongo.WriteException{WriteErrors: mongo.WriteErrors{{Code: 2, Message: dsn}}}),
	}
	for name, err := range cases {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(err.Error(), dsn) {
				t.Fatal("test error does not carry the value it should withhold")
			}
			if s := fmt.Sprint(errorAttrs(err)...); strings.Contains(s, "pw-") {
				t.Error("attributes carry the server message")
			}
		})
	}
	plain := errors.New("context deadline exceeded")
	if attrs := errorAttrs(plain); len(attrs) != 2 || attrs[1] != plain {
		t.Errorf("attrs = %v, want the error itself for a client-side error", attrs)
	}
}

func TestMongoWritesDropTsConn(t *testing.T) {
	useTestMongo(t)
	repo := &MongoRepo{}
	ctx := context.Background()
	in := lib.Pipeline{Id: "p-1", UserId: "user-a", Operators: []lib.Operator{
		{Id: "o-1", Config: map[string]string{tsConnKey: fakeDSN(), "window": "5"}},
		{Id: "o-2"},
	}}
	if err := repo.InsertPipeline(ctx, in); err != nil {
		t.Fatal(err)
	}
	checkConfigs(t, "after insert", storedConfigs(t, "p-1"), map[string]string{"window": "5"}, nil)
	if _, ok := in.Operators[0].Config[tsConnKey]; !ok {
		t.Error("insert modified the caller's config")
	}

	in.Operators[1].Config = map[string]string{tsConnKey: fakeDSN(), "topic": "t"}
	if err := repo.UpdatePipeline(ctx, in, "user-a"); err != nil {
		t.Fatal(err)
	}
	checkConfigs(t, "after update", storedConfigs(t, "p-1"),
		map[string]string{"window": "5"}, map[string]string{"topic": "t"})
}

func TestMongoReadsDropStoredTsConn(t *testing.T) {
	useTestMongo(t)
	repo := &MongoRepo{}
	ctx := context.Background()
	// Inserted past the repository, the way documents from before the fix are stored.
	for _, id := range []string{"p-1", "p-2"} {
		p := lib.Pipeline{Id: id, UserId: "user-a", Operators: []lib.Operator{
			{Id: "o-1", Config: map[string]string{tsConnKey: fakeDSN(), "window": "5"}},
			{Id: "o-2", Config: map[string]string{"topic": "t"}},
		}}
		if _, err := Mongo().InsertOne(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	want := []map[string]string{{"window": "5"}, {"topic": "t"}}

	one, err := repo.FindPipeline(ctx, "p-1", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	checkConfigs(t, "FindPipeline", []map[string]string{one.Operators[0].Config, one.Operators[1].Config}, want...)

	for _, admin := range []bool{false, true} {
		all, err := repo.All(ctx, "user-a", admin, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(all.Data) != 2 {
			t.Fatalf("All(admin=%v) returned %d pipelines, want 2", admin, len(all.Data))
		}
		for _, p := range all.Data {
			checkConfigs(t, fmt.Sprintf("All(admin=%v) %s", admin, p.Id),
				[]map[string]string{p.Operators[0].Config, p.Operators[1].Config}, want...)
		}
	}
}

func TestMongoRemoveStoredTsConn(t *testing.T) {
	useTestMongo(t)
	ctx := context.Background()
	dsn := fakeDSN()
	pipelines := []lib.Pipeline{
		{Id: "with", Operators: []lib.Operator{
			{Id: "o-1", Config: map[string]string{tsConnKey: dsn, "window": "5"}},
			{Id: "o-2"}, // config stored as null, which $[] has to step over
			{Id: "o-3", Config: map[string]string{"topic": "t"}},
			{Id: "o-4", Config: map[string]string{tsConnKey: dsn}},
		}},
		{Id: "without", Operators: []lib.Operator{{Id: "o-1", Config: map[string]string{"window": "5"}}}},
		{Id: "no operators"},
	}
	for _, p := range pipelines {
		if _, err := Mongo().InsertOne(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	raw := func(id string) bson.Raw {
		r, err := Mongo().FindOne(ctx, bson.M{"id": id}).Raw()
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	untouched := map[string]bson.Raw{"without": raw("without"), "no operators": raw("no operators")}

	log := captureLog(t)
	RemoveStoredTsConn(ctx, 20*time.Second)

	checkConfigs(t, "cleaned", storedConfigs(t, "with"),
		map[string]string{"window": "5"}, nil, map[string]string{"topic": "t"}, map[string]string{})
	for id, before := range untouched {
		if !bytes.Equal(raw(id), before) {
			t.Errorf("%q changed", id)
		}
	}
	if strings.Contains(log.String(), "pw-") {
		t.Error("log carries the DSN")
	}
	var entry struct {
		Msg      string `json:"msg"`
		Matched  int64  `json:"matched"`
		Modified int64  `json:"modified"`
	}
	if err := json.Unmarshal(log.Bytes(), &entry); err != nil {
		t.Fatalf("log is not one JSON line: %v", err)
	}
	if entry.Matched != 1 || entry.Modified != 1 {
		t.Errorf("first run logged matched %d, modified %d, want 1 and 1", entry.Matched, entry.Modified)
	}

	matched, modified, err := removeStoredTsConn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if matched != 0 || modified != 0 {
		t.Errorf("second run matched %d, modified %d, want 0 and 0", matched, modified)
	}
}

// A document the update cannot apply to makes the server quote it in the error, DSN included.
func TestMongoRemoveStoredTsConnFailureLogsNoValue(t *testing.T) {
	useTestMongo(t)
	ctx := context.Background()
	dsn := fakeDSN()
	if _, err := Mongo().InsertOne(ctx, bson.M{"id": "odd", "operators": bson.M{"config": bson.M{tsConnKey: dsn}}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := removeStoredTsConn(ctx); err == nil {
		t.Fatal("expected the update to fail on a non-array operators field")
	} else if !strings.Contains(err.Error(), "pw-") {
		t.Log("server did not quote the document; the check below still guards the log")
	}

	log := captureLog(t)
	RemoveStoredTsConn(ctx, 20*time.Second)
	if strings.Contains(log.String(), "pw-") {
		t.Error("log carries the DSN")
	}
	if !strings.Contains(log.String(), "could not remove ts_conn") || !strings.Contains(log.String(), "write_error_codes") {
		t.Error("failure not logged with its codes")
	}
}
