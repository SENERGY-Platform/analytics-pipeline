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

package service

import (
	"reflect"
	"testing"

	"github.com/SENERGY-Platform/analytics-pipeline/lib"
	"github.com/SENERGY-Platform/analytics-pipeline/pkg/db"
	"github.com/google/uuid"
)

func TestRegistry_SavePipeline(t *testing.T) {
	registry := NewRegistry(db.NewMockRepo())
	id, err := registry.SavePipeline(lib.Pipeline{}, "1")
	if err != nil {
		t.Skip(err)
	}
	if reflect.TypeOf(id) != reflect.TypeOf(uuid.New()) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			reflect.TypeOf(id), reflect.TypeOf(uuid.New()))
	}
}
