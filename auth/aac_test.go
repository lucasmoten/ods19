package auth_test

import (
	"path/filepath"
	"strings"
	"testing"

	"decipher.com/object-drive-server/auth"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/services/aac"
)

func newAACAuth(t *testing.T) auth.AACAuth {
	// These tests are dependent upon zookeeper, which the AAC will announce to.
	// Find the host + port, given a zookeeper node + zookeeper host + port
	// depend on a mapped hostname in /etc/hosts to find and connect zookeeper

	// The docker node name for zookeeper that denotes the zookeeper cluster
	// that zk is assumed to announce to in our test environment is hardcoded
	// here as 'zk'
	t.Logf("Discovering Zookeeper")
	aacHost := "aac"
	aacPort := 9093

	// AAC trust, client public & private key
	trustPath := filepath.Join("..", "defaultcerts", "clients", "client.trust.pem")
	certPath := filepath.Join("..", "defaultcerts", "clients", "test_1.cert.pem")
	keyPath := filepath.Join("..", "defaultcerts", "clients", "test_1.key.pem")

	t.Logf("AAC client initializing with trust: %s, cert: %s, key: %s", trustPath, certPath, keyPath)
	aacClient, err := aac.GetAACClient(aacHost, aacPort, trustPath, certPath, keyPath)
	if err != nil {
		t.Logf("Error getting AAC Client %s", err.Error())
		t.FailNow()
	}
	aacAuth := auth.AACAuth{Logger: config.RootLogger, Service: aacClient}
	t.Logf("AAC client ready")
	return aacAuth
}

type testAACAuth struct {
	subtestname          string
	userIdentity         string
	acm                  string
	permissions          []models.ODObjectPermission
	expectedFlattened    string
	expectedModified     string
	expectedIsAuthorized bool
	expectedIsError      bool
	expectedError        error
	expectedSnippets     string
}

func TestAACAuthGetFlattenedACM(t *testing.T) {
	aacAuth := newAACAuth(t)

	subtests := []testAACAuth{}
	subtests = append(subtests, testAACAuth{subtestname: "No ACM", expectedIsError: true, expectedError: auth.ErrACMNotSpecified})
	subtests = append(subtests, testAACAuth{subtestname: "Valid ACM", expectedIsError: false, acm: `{"version":"2.1.0", "classif": "U"}`})
	subtests = append(subtests, testAACAuth{subtestname: "Valid ACM w/Expected", expectedIsError: false, acm: `{"version":"2.1.0", "classif":"U"}`, expectedFlattened: `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_sar_id":[],"f_atom_energy":[],"f_macs":[]}`})
	subtests = append(subtests, testAACAuth{subtestname: "Invalid ACM", expectedIsError: true, acm: "meh", expectedError: auth.ErrServiceNotSuccessful})

	for testIdx, subtest := range subtests {
		t.Logf("Subtest %d: %s", testIdx, subtest.subtestname)
		flattenedACM, err := aacAuth.GetFlattenedACM(subtest.acm)
		// If expecting an error but didn't get one
		if subtest.expectedIsError && err == nil {
			if subtest.expectedError != nil {
				t.Logf("[x] Expected error (%s), got none", subtest.expectedError.Error())
			} else {
				t.Logf("[x] Expected error but got none")
			}
			t.Fail()
			continue
		}
		// If expecting an error, but got a different one
		if subtest.expectedIsError && err != nil && !strings.HasPrefix(err.Error(), subtest.expectedError.Error()) {
			t.Logf("[x] Expected error (%s) but got error (%s)", subtest.expectedError.Error(), err.Error())
			t.Fail()
			continue
		}
		// If not expecting an error, but got one
		if !subtest.expectedIsError && err != nil {
			t.Logf("[x] Got an error %s", err.Error())
			t.Fail()
			continue
		}
		// If the flattenedACM isn't what we expected
		if len(subtest.expectedFlattened) > 0 {
			if strings.Compare(subtest.expectedFlattened, flattenedACM) != 0 {
				t.Logf("[x] ACM returned did not match expected result.")
				t.Logf("%s", flattenedACM)
				t.Fail()
				continue
			} else {
				t.Logf("OK  Flattened ACM matches expected value")
			}
		}
		t.Logf("OK  All checks passed")
	}
}

func TestAACAuthGetSnippetsForUser(t *testing.T) {
	aacAuth := newAACAuth(t)

	subtests := []testAACAuth{}
	subtests = append(subtests, testAACAuth{expectedIsError: true, subtestname: "No User", expectedError: auth.ErrUserNotSpecified})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "Fake User", userIdentity: "Fake User"})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "Jonathan Holmes", userIdentity: "CN=Holmes Jonathan,OU=People,OU=Bedrock,OU=Six 3 Systems,O=U.S. Government,C=US"})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "Tester 10", userIdentity: "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us", expectedSnippets: "dissem_countries allowed (USA) AND f_accms disallow () AND f_atom_energy allowed () AND f_clearance allowed (ts,s,c,u) AND f_macs disallow (tide,bir,watchdog) AND f_missions allowed () AND f_oc_org allowed (dia) AND f_regions allowed () AND f_sar_id allowed () AND f_sci_ctrls disallow (hcs_p,kdk,rsv) AND f_share allowed (dctc_odrive,dctc_odrive_g1,cntesttester10oupeopleoudaeouchimeraou_s_governmentcus,cusou_s_governmentouchimeraoudaeoupeoplecntesttester10)"})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "Uppercase Tester 10", userIdentity: "CN=Test Tester10,OU=People,OU=DAE,OU=Chimera,O=U.S. Government,C=US", expectedSnippets: "dissem_countries allowed (USA) AND f_accms disallow () AND f_atom_energy allowed () AND f_clearance allowed (ts,s,c,u) AND f_macs disallow (tide,bir,watchdog) AND f_missions allowed () AND f_oc_org allowed (dia) AND f_regions allowed () AND f_sar_id allowed () AND f_sci_ctrls disallow (hcs_p,kdk,rsv) AND f_share allowed (dctc_odrive,dctc_odrive_g1,cntesttester10oupeopleoudaeouchimeraou_s_governmentcus,cusou_s_governmentouchimeraoudaeoupeoplecntesttester10)"})

	for testIdx, subtest := range subtests {
		t.Logf("Subtest %d: %s", testIdx, subtest.subtestname)
		snippets, err := aacAuth.GetSnippetsForUser(subtest.userIdentity)
		// If expecting an error but didn't get one
		if subtest.expectedIsError && err == nil {
			if subtest.expectedError != nil {
				t.Logf("[x] Expected error (%s), got none", subtest.expectedError.Error())
			} else {
				t.Logf("[x] Expected error but got none")
			}
			t.Fail()
			continue
		}
		// If expecting an error, but got a different one
		if subtest.expectedIsError && err != nil && !strings.HasPrefix(err.Error(), subtest.expectedError.Error()) {
			t.Logf("[x] Expected error (%s) but got error (%s)", subtest.expectedError.Error(), err.Error())
			t.Fail()
			continue
		}
		// If not expecting an error, but got one
		if !subtest.expectedIsError && err != nil {
			t.Logf("[x] Got an error %s", err.Error())
			t.Fail()
			continue
		}

		// If the snippets isn't what we expected
		if len(subtest.expectedSnippets) != 0 {
			// If the length of snippets is empty
			if snippets == nil {
				t.Logf("[x] No snippets were returned")
				t.Fail()
				continue
			}
			actualSnippets := snippets.String()
			if strings.Compare(actualSnippets, subtest.expectedSnippets) != 0 {
				t.Logf("[x] Expected snippets to be %s but got %s", subtest.expectedSnippets, actualSnippets)
				t.Fail()
				continue
			}
		}
		t.Logf("OK  All checks passed")
	}
}

func TestAACAuthInjectPermissionsIntoACM(t *testing.T) {

	aacAuth := newAACAuth(t)
	acmUnclassEveryone := `{"classif":"U","share":null,"version":"2.1.0"}`
	acmUnclassODrive := `{"classif":"U","share":{"projects":{"dctc":{"disp_nm":"DCTC","groups":["ODrive"]}}},"version":"2.1.0"}`
	acmUnclassODriveT1T2 := `{"classif":"U","share":{"projects":{"dctc":{"disp_nm":"DCTC","groups":["ODrive"]}},"users":["cnt1","cnt2"]},"version":"2.1.0"}`
	acmUnclassT1T2 := `{"classif":"U","share":{"users":["cnt1","cnt2"]},"version":"2.1.0"}`
	T1CRUDS := models.ODObjectPermission{
		Grantee: "cnt1",
		AcmGrantee: models.ODAcmGrantee{
			Grantee:               "cnt1",
			UserDistinguishedName: models.ToNullString("cn=t1"),
			DisplayName:           models.ToNullString("t1")},
		AcmShare:    `{"users":["cnt1"]}`,
		AllowCreate: true, AllowRead: true, AllowUpdate: true, AllowDelete: true, AllowShare: true}
	T2R := models.ODObjectPermission{
		Grantee: "cnt2",
		AcmGrantee: models.ODAcmGrantee{
			Grantee:               "cnt2",
			UserDistinguishedName: models.ToNullString("cn=t2"),
			DisplayName:           models.ToNullString("t2")},
		AcmShare:    `{"users":["cnt2"]}`,
		AllowCreate: false, AllowRead: true, AllowUpdate: false, AllowDelete: false, AllowShare: false}
	permissionsT1T2 := []models.ODObjectPermission{T1CRUDS, T2R}
	nopermissions := []models.ODObjectPermission{}

	subtests := []testAACAuth{}
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "shared + no permissions unmodified", permissions: nopermissions, acm: acmUnclassODrive, expectedModified: acmUnclassODrive})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "shared + permissions changes", permissions: permissionsT1T2, acm: acmUnclassODrive, expectedModified: acmUnclassODriveT1T2})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "public + permissions changes", permissions: permissionsT1T2, acm: acmUnclassEveryone, expectedModified: acmUnclassT1T2})

	for testIdx, subtest := range subtests {
		t.Logf("Subtest %d: %s", testIdx, subtest.subtestname)
		modifiedAcm, err := aacAuth.InjectPermissionsIntoACM(subtest.permissions, subtest.acm)
		// If expecting an error but didn't get one
		if subtest.expectedIsError && err == nil {
			if subtest.expectedError != nil {
				t.Logf("[x] Expected error (%s), got none", subtest.expectedError.Error())
			} else {
				t.Logf("[x] Expected error but got none")
			}
			t.Fail()
			continue
		}
		// If expecting an error, but got a different one
		if subtest.expectedIsError && err != nil && !strings.HasPrefix(err.Error(), subtest.expectedError.Error()) {
			t.Logf("[x] Expected error (%s) but got error (%s)", subtest.expectedError.Error(), err.Error())
			t.Fail()
			continue
		}
		// If not expecting an error, but got one
		if !subtest.expectedIsError && err != nil {
			t.Logf("[x] Got an error %s", err.Error())
			t.Fail()
			continue
		}
		// If the result isnt what we expect
		if subtest.expectedModified != modifiedAcm {
			t.Logf("[x] Expected %s but got %s for injecting permissions", subtest.expectedModified, modifiedAcm)
			t.Fail()
			continue
		}
		t.Logf("OK  All checks passed")
	}
}

func TestAACAuthIsUserAuthorizedForACM(t *testing.T) {
	aacAuth := newAACAuth(t)

	acmUnclass := `{"version":"2.1.0","classif":"U","share":{"projects":{"dctc":{"disp_nm":"DCTC","groups":["ODrive"]}}}}`
	acmSecret := `{"version":"2.1.0","classif":"S","banner":"SECRET","share":{"projects":{"dctc":{"disp_nm":"DCTC","groups":["ODrive"]}}}}`
	acmTopSecret := `{"version":"2.1.0","classif":"TS","banner":"TOP SECRET","share":{"projects":{"dctc":{"disp_nm":"DCTC","groups":["ODrive"]}}}}`
	acmODriveG1 := `{"version":"2.1.0","classif":"U","share":{"projects":{"dctc":{"disp_nm":"DCTC","groups":["ODrive_G1"]}}}}`
	acmODriveG2 := `{"version":"2.1.0","classif":"U","share":{"projects":{"dctc":{"disp_nm":"DCTC","groups":["ODrive_G2"]}}}}`

	idFake := "Fake User"
	idJon := "CN=Holmes Jonathan,OU=People,OU=Bedrock,OU=Six 3 Systems,O=U.S. Government,C=US"
	idDave := "CN=Yantz David,OU=People,OU=Bedrock,OU=Mantech,O=U.S. Government,C=US"
	idTester10 := "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	idTester01 := "cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"

	subtests := []testAACAuth{}
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "No User or ACM expect error", expectedError: auth.ErrACMNotSpecified})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "No User expect error", expectedError: auth.ErrUserNotSpecified, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "User with Invalid ACM expect error", userIdentity: idFake, expectedError: auth.ErrACMNotValid, acm: `{"x":"123"}`})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Fake User (-DIAS) for unclass expect error", userIdentity: idFake, expectedError: auth.ErrUserNotAuthorized, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Jonathan Holmes (-DIAS) for unclass expect error", userIdentity: idJon, expectedError: auth.ErrUserNotAuthorized, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "David Yantz (+DIAS) for unclass expect error", userIdentity: idDave, expectedError: auth.ErrUserNotAuthorized, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Tester 10 (+DIAS) for unclass", userIdentity: idTester10, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Tester 01 (+DIAS) for unclass", userIdentity: idTester01, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Fake User (-DIAS) for secret expect error", userIdentity: idFake, expectedError: auth.ErrUserNotAuthorized, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Jonathan Holmes (-DIAS) for secret expect error", userIdentity: idJon, expectedError: auth.ErrUserNotAuthorized, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "David Yantz (+DIAS) for secret expect error", userIdentity: idDave, expectedError: auth.ErrUserNotAuthorized, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Tester 10 (+DIAS) for secret", userIdentity: idTester10, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Tester 01 (+DIAS) for secret expect error", userIdentity: idTester01, expectedError: auth.ErrUserNotAuthorized, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Fake User (-DIAS) for top secret expect error", userIdentity: idFake, expectedError: auth.ErrUserNotAuthorized, acm: acmTopSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Jonathan Holmes (-DIAS) for top secret expect error", userIdentity: idJon, expectedError: auth.ErrUserNotAuthorized, acm: acmTopSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "David Yantz (+DIAS) for top secret expect error", userIdentity: idDave, expectedError: auth.ErrUserNotAuthorized, acm: acmTopSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Tester 10 (+DIAS) for top secret", userIdentity: idTester10, acm: acmTopSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Tester 01 (+DIAS) for top secret expect error", userIdentity: idTester01, expectedError: auth.ErrUserNotAuthorized, acm: acmTopSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Tester 10 (+DIAS) for Odrive_G1", userIdentity: idTester10, acm: acmODriveG1})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Tester 10 (+DIAS) for Odrive_G2 expect error", userIdentity: idTester10, expectedError: auth.ErrUserNotAuthorized, acm: acmODriveG2})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Tester 01 (+DIAS) for Odrive_G1 expect error", userIdentity: idTester01, expectedError: auth.ErrUserNotAuthorized, acm: acmODriveG1})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Tester 01 (+DIAS) for Odrive_G2", userIdentity: idTester01, acm: acmODriveG2})

	for testIdx, subtest := range subtests {
		t.Logf("Subtest %d: %s", testIdx, subtest.subtestname)
		authorized, err := aacAuth.IsUserAuthorizedForACM(subtest.userIdentity, subtest.acm)
		// If expecting an error but didn't get one
		if subtest.expectedIsError && err == nil {
			if subtest.expectedError != nil {
				t.Logf("[x] Expected error (%s), got none", subtest.expectedError.Error())
			} else {
				t.Logf("[x] Expected error but got none")
			}
			t.Fail()
			continue
		}
		// If expecting an error, but got a different one
		if subtest.expectedIsError && err != nil && !strings.HasPrefix(err.Error(), subtest.expectedError.Error()) {
			t.Logf("[x] Expected error (%s) but got error (%s)", subtest.expectedError.Error(), err.Error())
			t.Fail()
			continue
		}
		// If not expecting an error, but got one
		if !subtest.expectedIsError && err != nil {
			t.Logf("[x] Got an error %s", err.Error())
			t.Fail()
			continue
		}
		// If the result isnt what we expect
		if subtest.expectedIsAuthorized != authorized {
			t.Logf("[x] Expected %t but got %t for authorized", subtest.expectedIsAuthorized, authorized)
			t.Fail()
			continue
		}
		t.Logf("OK  All checks passed")
	}
}
