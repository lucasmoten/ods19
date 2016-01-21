package dao_test

import (
	"testing"

	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
)

func TestGetSanitizedPageNumber(t *testing.T) {
	if dao.GetSanitizedPageNumber(-1) != 1 {
		t.Error("expected 1")
	}
	if dao.GetSanitizedPageNumber(0) != 1 {
		t.Error("expected 1")
	}
	if dao.GetSanitizedPageNumber(1) != 1 {
		t.Error("expected 1")
	}
	if dao.GetSanitizedPageNumber(100) != 100 {
		t.Failed()
	}
}
