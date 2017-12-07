package protocol_test

import (
	"testing"

	"github.com/deciphernow/object-drive-server/protocol"
)

func TestWithCallerPermission(t *testing.T) {

	tester09dn := "cntesttester09oupeopleoudaeouchimeraou_s_governmentcus"
	utester09dn := "user/cntesttester09oupeopleoudaeouchimeraou_s_governmentcus"
	utester10dn := "user/cntesttester10oupeopleoudaeouchimeraou_s_governmentcus"
	geveryone := "group/_everyone"

	cases := []struct {
		callerDN      string
		description   string
		perm          protocol.Permission
		C, R, U, D, S bool
	}{
		{
			callerDN:    tester09dn,
			description: "Explicit read only",
			perm:        protocol.Permission{Read: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}},
			C:           false, R: true, U: false, D: false, S: false,
		},
		{
			callerDN:    tester09dn,
			description: "Owner full cruds, second user has read",
			perm:        protocol.Permission{Create: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}, Read: protocol.PermissionCapability{AllowedResources: []string{utester09dn, utester10dn}}, Update: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}, Delete: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}, Share: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}},
			C:           true, R: true, U: true, D: true, S: true,
		},
		{
			callerDN:    tester09dn,
			description: "Explicit create, read, update",
			perm:        protocol.Permission{Create: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}, Read: protocol.PermissionCapability{AllowedResources: []string{utester09dn, utester10dn}}, Update: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}},
			C:           true, R: true, U: true, D: false, S: false,
		},
		{
			callerDN:    tester09dn,
			description: "Owner CUDS + Everyone R = CRUDS",
			perm:        protocol.Permission{Create: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}, Read: protocol.PermissionCapability{AllowedResources: []string{geveryone}}, Update: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}, Delete: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}, Share: protocol.PermissionCapability{AllowedResources: []string{utester09dn}}},
			C:           true, R: true, U: true, D: true, S: true,
		},
	}

	for _, c := range cases {
		var obj protocol.Object
		obj.Permission = c.perm
		// Assert that WithCallerPermission is implemented correctly.
		obj = obj.WithCallerPermission(protocol.Caller{DistinguishedName: c.callerDN})
		cp := obj.CallerPermission
		if c.C != cp.AllowCreate || c.R != cp.AllowRead || c.U != cp.AllowUpdate || c.D != cp.AllowDelete || c.S != cp.AllowShare {
			t.Logf("Failed for %s", c.description)
			template := "expected create=%v read=%v update=%v delete=%v share=%v but got %s"
			t.Errorf(template, c.C, c.R, c.U, c.D, c.S, cp)
		}
	}
}
