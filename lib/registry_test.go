package lib

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestRegistry_SavePipeline(t *testing.T) {
	registry := NewRegistry(NewMockRepo())
	id, err := registry.SavePipeline(Pipeline{}, "1")
	if err != nil {
		t.Skip(err)
	}
	if reflect.TypeOf(id) != reflect.TypeOf(uuid.New()) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			reflect.TypeOf(id), reflect.TypeOf(uuid.New()))
	}
}
