package models_test

import (
	"encoding/json"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
)

func TestNullString(t *testing.T) {

	type test struct {
		Name models.NullString
		Age  int
	}

	jsonData := `{ "Name": "", "Age": 40 }`
	b := []byte(jsonData)

	var obj1 test
	err := json.Unmarshal(b, &obj1)
	if err != nil {
		t.Fail()
	}
	if obj1.Name.Valid != false {
		t.Errorf("Expected valid to be false. Got: %v\n", obj1.Name.Valid)
	}
	if v, _ := obj1.Name.Value(); v != nil {
		t.Errorf("Expected Value() to return nil for field Name when given: %s\n", jsonData)
	}

}
