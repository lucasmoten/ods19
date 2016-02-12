package dao

import (
	"testing"

	"decipher.com/oduploader/metadata/models"
)

func TestGetObject(t *testing.T) {

	expected := &models.ODObject{Name: "FOO"}
	fake := &FakeDAO{Object: expected}
	result, _ := fake.GetObject(expected, false)

	if expected.Name != result.Name {
		t.Fail()
	}
}
