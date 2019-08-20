package lib

import (
	"reflect"
	"testing"

	uuid "github.com/satori/go.uuid"
)

func TestRegistry_SavePipeline(t *testing.T) {
	registry := NewRegistry(NewMockRepo())
	id := registry.SavePipeline(Pipeline{}, "1")
	if reflect.TypeOf(id) != reflect.TypeOf(uuid.NewV4()) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			reflect.TypeOf(id), reflect.TypeOf(uuid.NewV4()))
	}
}
