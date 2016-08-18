package protocol_test

import (
	"testing"

	"decipher.com/object-drive-server/protocol"
)

func TestWithRolledUp(t *testing.T) {

	tester09dn := "cntesttester09oupeopleoudaeouchimeraou_s_governmentcus"
	tester10dn := "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus"

	cases := []struct {
		callerDN      string
		perms         []protocol.Permission
		C, R, U, D, S bool
	}{
		{
			callerDN: tester09dn,
			perms: []protocol.Permission{
				{Grantee: tester09dn, AllowRead: true},
			},
			C: false, R: true, U: false, D: false, S: false,
		},
		{
			callerDN: tester09dn,
			perms: []protocol.Permission{
				{Grantee: tester09dn, AllowCreate: true, AllowRead: true, AllowUpdate: true, AllowDelete: true, AllowShare: true},
				{Grantee: tester10dn, AllowRead: true},
			},
			C: true, R: true, U: true, D: true, S: true,
		},
		{
			callerDN: tester09dn,
			perms: []protocol.Permission{
				{Grantee: tester09dn, AllowCreate: true, AllowRead: true, AllowUpdate: true},
				{Grantee: tester10dn, AllowRead: true},
			},
			C: true, R: true, U: true, D: false, S: false,
		},
	}

	for _, c := range cases {
		var cp protocol.CallerPermission
		// Assert that WithRolledUp is implemented correctly.
		cp = cp.WithRolledUp(protocol.Caller{DistinguishedName: c.callerDN}, c.perms...)
		if c.C != cp.AllowCreate || c.R != cp.AllowRead || c.U != cp.AllowUpdate || c.D != cp.AllowDelete || c.S != cp.AllowShare {
			template := "expected create=%v read=%v update=%v delete=%v share=%v but got %s"
			t.Errorf(template, c.C, c.R, c.U, c.D, c.S)
		}
	}
}
