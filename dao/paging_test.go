package dao_test

import (
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/dao"
)

func TestDAOGetSanitizedPageNumber(t *testing.T) {
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

func TestDAOGetSanitizedPageSize(t *testing.T) {
	if dao.GetSanitizedPageSize(-1) != 1 {
		t.Error("expected 1")
	}
	if dao.GetSanitizedPageSize(0) != 1 {
		t.Error("expected 1")
	}
	if dao.GetSanitizedPageSize(1) != 1 {
		t.Error("expected 1")
	}
	if dao.GetSanitizedPageSize(100) != 100 {
		t.Error("expected 100")
	}
	if dao.GetSanitizedPageSize(10000) != 10000 {
		t.Error("expected 10000")
	}
	if dao.GetSanitizedPageSize(100000) != 10000 {
		t.Error("expected 10000")
	}
}

func TestDAOGetLimit(t *testing.T) {
	if dao.GetLimit(0, 0) != 1 {
		t.Error("expected 1")
	}
	if dao.GetLimit(1, 100) != 100 {
		t.Error("expected 100")
	}
	if dao.GetLimit(1, 100000) != 10000 {
		t.Error("expected 10000")
	}
	if dao.GetLimit(-1, 0) != 1 {
		t.Error("expected 1")
	}
}

func TestDAOGetOffset(t *testing.T) {
	if dao.GetOffset(1, 20) != 0 {
		t.Error("expected 1")
	}
	if dao.GetOffset(3, 20) != 40 {
		t.Error("expected 41")
	}
	if dao.GetOffset(5, 100) != 400 {
		t.Error("expected 401")
	}
	if dao.GetOffset(0, 99) != 0 {
		t.Error("expected 1")
	}
}

func TestDAOGetPageCount(t *testing.T) {
	if dao.GetPageCount(34, 20) != 2 {
		t.Error("expected 2")
	}
	if dao.GetPageCount(8000, 20) != 400 {
		t.Error("expected 400")
	}
}
