package dao_test

import (
	"bytes"
	"testing"
)

func TestGetParents(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	parent, child, err := createParentChildObjectPair(usernames[1])
	if err != nil {
		t.Errorf("could not create parent-child object pair: %v\n", err)
	}

	parent, err = d.CreateObject(&parent)
	if err != nil {
		t.Errorf("could not create parent in database: %v\n", err)
	}
	child, err = d.CreateObject(&child)
	if err != nil {
		t.Errorf("could not create child in database: %v\n", err)
	}

	parents, err := d.GetParents(child)
	if err != nil {
		t.Errorf("could not get parents for child %v\n", err)
	}

	if len(parents) != 1 {
		t.Errorf("expected exactly one parent to be returned, got %v\n", len(parents))
	}

	if bytes.Compare(parents[0].ID, parent.ID) != 0 {
		t.Errorf("expected parent IDs to be the same")
	}
}
