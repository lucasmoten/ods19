package protocol_test

import (
	"testing"

	"decipher.com/object-drive-server/protocol"
)

func TestWithRolledUp(t *testing.T) {

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
		var cp protocol.CallerPermission
		// Assert that WithRolledUp is implemented correctly.
		cp = cp.WithRolledUp(protocol.Caller{DistinguishedName: c.callerDN}, c.perm)
		if c.C != cp.AllowCreate || c.R != cp.AllowRead || c.U != cp.AllowUpdate || c.D != cp.AllowDelete || c.S != cp.AllowShare {
			t.Logf("Failed for %s", c.description)
			template := "expected create=%v read=%v update=%v delete=%v share=%v but got %s"
			t.Errorf(template, c.C, c.R, c.U, c.D, c.S, cp)
		}
	}
}

func TestFlattenedUserFromResource(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "user/xyz",
			expected: "xyz",
		},
		{
			input:    "user/cntesttester09oupeopleoudaeouchimeraou_s_governmentcus",
			expected: "cntesttester09oupeopleoudaeouchimeraou_s_governmentcus",
		},
		{
			input:    "group/it doesnt matter",
			expected: "group",
		},
		{
			input:    "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10",
			expected: "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
		},
	}
	for i, c := range cases {
		r := protocol.GetFlattenedUserFromResource(c.input)
		if c.expected != r {
			template := "%d expected %s but got %s"
			t.Errorf(template, i, c.expected, r)
		}
	}
}

func TestFlattenedGroupFromResource(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "user/xyz",
			expected: "user",
		},
		{
			input:    "user/cntesttester09oupeopleoudaeouchimeraou_s_governmentcus",
			expected: "user",
		},
		{
			input:    "group/it doesnt matter",
			expected: "itdoesntmatter",
		},
		{
			input:    "group/dctc/DCTC/ODrive/DCTC ODrive",
			expected: "dctc_odrive",
		},
	}
	for i, c := range cases {
		r := protocol.GetFlattenedGroupFromResource(c.input)
		if c.expected != r {
			template := "%d expected %s but got %s"
			t.Errorf(template, i, c.expected, r)
		}
	}
}
