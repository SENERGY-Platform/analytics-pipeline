package lib

import (
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestRegistry_SavePipeline(t *testing.T) {
	registry := NewRegistry(NewMockRepo())
	id, err := registry.SavePipeline(Pipeline{}, "1")
	if err != nil {
		t.Errorf(err.Error())
	}
	if reflect.TypeOf(id) != reflect.TypeOf(uuid.New()) {
		t.Errorf("handler returned unexpected body: got %v want %v",
			reflect.TypeOf(id), reflect.TypeOf(uuid.New()))
	}
}
