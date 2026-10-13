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
	"errors"
	"maps"
	"time"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// tsConnKey is the operator config key under which flow engines sent the Timescale DSN,
// password included; it is never stored or returned.
const tsConnKey = "ts_conn"

// withoutTsConn returns pipeline with ts_conn removed from every operator config. Operators
// and configs are copied, so the caller's pipeline keeps its maps unchanged.
func withoutTsConn(pipeline lib.Pipeline) lib.Pipeline {
	if pipeline.Operators == nil {
		return pipeline
	}
	operators := make([]lib.Operator, len(pipeline.Operators))
	for i, operator := range pipeline.Operators {
		if _, ok := operator.Config[tsConnKey]; ok {
			operator.Config = maps.Clone(operator.Config)
			delete(operator.Config, tsConnKey)
		}
		operators[i] = operator
	}
	pipeline.Operators = operators
	return pipeline
}

// removeStoredTsConn unsets ts_conn in the operator configs of all stored pipelines; a second
// run matches nothing. The all-positional operator $[] needs MongoDB 3.6 or later.
func removeStoredTsConn(ctx context.Context) (matched, modified int64, err error) {
	res, err := Mongo().UpdateMany(ctx,
		bson.M{"operators.config." + tsConnKey: bson.M{"$exists": true}},
		bson.M{"$unset": bson.M{"operators.$[].config." + tsConnKey: ""}},
	)
	if err != nil {
		return 0, 0, err
	}
	return res.MatchedCount, res.ModifiedCount, nil
}

// RemoveStoredTsConn cleans the stored pipelines and logs only the counts. A failure is logged
// and not returned, since the read path drops ts_conn from documents that still have it.
func RemoveStoredTsConn(ctx context.Context, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	matched, modified, err := removeStoredTsConn(ctx)
	if err != nil {
		util.Logger.ErrorContext(ctx, "could not remove ts_conn from stored pipelines", errorAttrs(err)...)
		return
	}
	util.Logger.InfoContext(ctx, "removed ts_conn from stored pipelines", "matched", matched, "modified", modified)
}

// errorAttrs describes err for the log without the server's message, which can quote the
// offending document and with it a DSN.
func errorAttrs(err error) []any {
	var we mongo.WriteException
	if errors.As(err, &we) {
		codes := make([]int, 0, len(we.WriteErrors))
		for _, e := range we.WriteErrors {
			codes = append(codes, e.Code)
		}
		attrs := []any{"write_error_codes", codes}
		if we.WriteConcernError != nil {
			attrs = append(attrs, "write_concern_error_code", we.WriteConcernError.Code)
		}
		return attrs
	}
	var ce mongo.CommandError
	if errors.As(err, &ce) {
		return []any{"error_code", ce.Code, "error_name", ce.Name}
	}
	var se mongo.ServerError
	if errors.As(err, &se) {
		return []any{"error", "mongo server error, message withheld"}
	}
	return []any{"error", err}
}
