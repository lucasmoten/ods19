package auth_test

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"bitbucket.di2e.net/dime/object-drive-server/auth"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/services/aac"
)

func getAACPort() int {
	// TODO: modify dockerhelp_test.go to be able to lookup environment variable values in container
	// will want to get COM_DECIPHERNOW_SERVER_CONFIG_THRIFT_PORT from Config.Env (get 9000)
	// Then, compare that to the exposed ports in HostConfig.PortBindings (map 9000 to 9093)
	aacPortOverride := os.Getenv("OD_AAC_TEST_PORT")
	if len(aacPortOverride) > 0 {
		p, err := strconv.Atoi(aacPortOverride)
		if err != nil {
			log.Printf("reading OD_AAC_TEST_PORT: %v", err)
		}
		return p
	}
	return 9093
}

func getAACAddress() string {
	return "localhost"
	//return "aac"
	container, err := getDockerContainerIDFromName("docker_aac_1")
	if err != nil {
		log.Printf("error getting containerid: %v", err)
		return "aac"
	}
	addr, err := getIPAddressForContainer(container)
	if err != nil {
		log.Printf("error getting address: %v", err)
		return "aac"
	}
	return addr
}

func newAACAuth(t *testing.T) auth.AACAuth {
	// AAC server and port hardcoded
	aacHost := getAACAddress()
	aacPort := getAACPort()
	// AAC trust, client public & private key
	trustPath := filepath.Join("..", "defaultcerts", "client-aac", "trust", "client.trust.pem")
	certPath := filepath.Join("..", "defaultcerts", "client-aac", "id", "client.cert.pem")
	keyPath := filepath.Join("..", "defaultcerts", "client-aac", "id", "client.key.pem")
	serverCN := "twl-server-generic2"

	t.Logf("AAC client initializing with trust: %s, cert: %s, key: %s", trustPath, certPath, keyPath)
	aacClient, err := aac.GetAACClient(aacHost, aacPort, trustPath, certPath, keyPath, serverCN)
	if err != nil {
		t.Logf("Error getting AAC Client %s", err.Error())
		t.FailNow()
	}
	aacAuth := auth.AACAuth{Logger: config.RootLogger, Service: aacClient}
	t.Logf("AAC client ready")
	return aacAuth
}

type testAACAuth struct {
	subtestname                 string
	userIdentity                string
	owner                       string
	acm                         string
	permissions                 []models.ODObjectPermission
	expectedFlattened           string
	expectedModifiedAcm         string
	expectedModifiedPermissions []models.ODObjectPermission
	expectedIsAuthorized        bool
	expectedIsError             bool
	expectedError               error
	expectedSnippets            []string
	expectedGroups              []string
	creating                    bool
	failed                      bool
}

func TestAACAuthGetFlattenedACM(t *testing.T) {
	aacAuth := newAACAuth(t)

	subtests := []testAACAuth{}
	subtests = append(subtests, testAACAuth{subtestname: "No ACM", expectedIsError: true, expectedError: auth.ErrACMNotSpecified})
	subtests = append(subtests, testAACAuth{subtestname: "Valid ACM", expectedIsError: false, acm: `{"version":"2.1.0", "classif": "U"}`})
	subtests = append(subtests, testAACAuth{subtestname: "Valid ACM w/Expected", expectedIsError: false, acm: `{"version":"2.1.0", "classif":"U"}`, expectedFlattened: `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_sar_id":[],"f_atom_energy":[],"f_macs":[]}`})
	subtests = append(subtests, testAACAuth{subtestname: "Invalid ACM", expectedIsError: true, acm: "meh", expectedError: auth.ErrACMResponseFailed})
	subtests = append(subtests, testAACAuth{subtestname: "CTE Security Service 54 without portion in input", expectedIsError: false, acm: `{"version":"2.1.0", "classif":"TS", "owner_prod":["USA"], "atom_energy":[], "sar_id":[], "sci_ctrls":["SI"], "disponly_to":[""], "dissem_ctrls": ["OC","NF"], "non_ic":[], "rel_to":[], "declass_ex": "50X1-HUM", "deriv_from": "FROM MULTIPLE SOURCES", "fgi_open": ["ISR"], "fgi_protect": [], "banner": "TOP SECRET//SI//FGI ISR//ORCON/NOFORN", "dissem_countries": ["USA"], "accms": [], "macs": [], "oc_attribs": [{"orgs":["DOD_DIA"],"missions":[],"regions":[]}], "share": {"users":[],"projects":{"bedrock":{"disp_nm":"Bedrock","groups":["WATCHDOG_USER"]}}},"disp_only":""}`, expectedFlattened: `{"version":"2.1.0","classif":"TS","owner_prod":["USA"],"atom_energy":[],"sar_id":[],"sci_ctrls":["SI"],"disponly_to":[""],"dissem_ctrls":["OC","NF"],"non_ic":[],"rel_to":[],"declass_ex":"50X1-HUM","deriv_from":"FROM MULTIPLE SOURCES","fgi_open":["ISR"],"fgi_protect":[],"portion":"TS//SI//FGI ISR//OC/NF","banner":"TOP SECRET//SI//FGI ISR//ORCON/NOFORN","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":["DOD_DIA"],"missions":[],"regions":[]}],"share":{"users":[],"projects":{"bedrock":{"disp_nm":"Bedrock","groups":["WATCHDOG_USER"]}}},"f_clearance":["ts"],"f_sci_ctrls":["si"],"f_accms":[],"f_oc_org":["dod_dia","dni"],"f_regions":[],"f_missions":[],"f_share":["bedrock_watchdog_user"],"f_sar_id":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`})
	subtests = append(subtests, testAACAuth{subtestname: "ACM has empty project in share", expectedIsError: false, acm: `{"version":"2.1.0", "classif":"U", "share":{"projects":{}}}`, expectedFlattened: `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"share":{"users":null,"projects":{}},"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_sar_id":[],"f_atom_energy":[],"f_macs":[]}`})
	subtests = append(subtests, testAACAuth{subtestname: "ACM has empty users in share", expectedIsError: false, acm: `{"version":"2.1.0", "classif":"U", "share":{"users":[]}}`, expectedFlattened: `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"share":{"users":[],"projects":null},"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_sar_id":[],"f_atom_energy":[],"f_macs":[]}`})
	subtests = append(subtests, testAACAuth{subtestname: "ACM has empty users and projects in share", expectedIsError: false, acm: `{"version":"2.1.0", "classif":"U", "share":{"users":[],"projects":{}}}`, expectedFlattened: `{"version":"2.1.0","classif":"U","portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"share":{"users":[],"projects":{}},"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_sar_id":[],"f_atom_energy":[],"f_macs":[]}`})
	// The following test fails because AAC is improperly parsing ISR from the portion and changing to Non-US
	//subtests = append(subtests, testAACAuth{subtestname: "CTE Security Service 54 with portion in input", expectedIsError: false, acm: `{"version":"2.1.0", "classif":"TS", "owner_prod":["USA"], "atom_energy":[], "sar_id":[], "sci_ctrls":["SI"], "disponly_to":[""], "dissem_ctrls": ["OC","NF"], "non_ic":[], "rel_to":[], "declass_ex": "50X1-HUM", "deriv_from": "FROM MULTIPLE SOURCES", "fgi_open": ["ISR"], "fgi_protect": [], "portion": "//ISR TS//SI//OC/NF", "banner": "TOP SECRET//SI//FGI ISR//ORCON/NOFORN", "dissem_countries": ["USA"], "accms": [], "macs": [], "oc_attribs": [{"orgs":["DOD_DIA"],"missions":[],"regions":[]}], "share": {"users":[],"projects":{"bedrock":{"disp_nm":"Bedrock","groups":["WATCHDOG_USER"]}}},"disp_only":""}`, expectedFlattened: `{"version":"2.1.0","classif":"TS","owner_prod":["USA"],"atom_energy":[],"sar_id":[],"sci_ctrls":["SI"],"disponly_to":[""],"dissem_ctrls":["OC","NF"],"non_ic":[],"rel_to":[],"declass_ex":"50X1-HUM","deriv_from":"FROM MULTIPLE SOURCES","fgi_open":["ISR"],"fgi_protect":[],"portion":"TS//SI//FGI ISR//OC/NF","banner":"TOP SECRET//SI//FGI ISR//ORCON/NOFORN","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":["DOD_DIA"],"missions":[],"regions":[]}],"share":{"users":[],"projects":{"bedrock":{"disp_nm":"Bedrock","groups":["WATCHDOG_USER"]}}},"f_clearance":["ts"],"f_sci_ctrls":["si"],"f_accms":[],"f_oc_org":["dod_dia","dni"],"f_regions":[],"f_missions":[],"f_share":["bedrock_watchdog_user"],"f_sar_id":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`})

	for testIdx, subtest := range subtests {
		t.Logf("Subtest %d: %s", testIdx, subtest.subtestname)
		flattenedACM, _, err := aacAuth.GetFlattenedACM(subtest.acm)
		// If expecting an error but didn't get one
		if subtest.expectedIsError && err == nil {
			subtest.failed = true
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
			subtest.failed = true
			t.Logf("[x] Expected error (%s) but got error (%s)", subtest.expectedError.Error(), err.Error())
			t.Fail()
			continue
		}
		// If not expecting an error, but got one
		if !subtest.expectedIsError && err != nil {
			subtest.failed = true
			t.Logf("[x] Got an error %s", err.Error())
			t.Fail()
			continue
		}
		// If the flattenedACM isn't what we expected
		if len(subtest.expectedFlattened) > 0 {
			if strings.Compare(subtest.expectedFlattened, flattenedACM) != 0 {
				subtest.failed = true
				t.Logf("[x] ACM returned did not match expected result.")
				t.Logf("\t Sent: %s", subtest.acm)
				t.Logf("\t Received: %s", flattenedACM)
				t.Logf("\t Expected: %s", subtest.expectedFlattened)
				t.Fail()
				continue
			} else {
				t.Logf("OK  Flattened ACM matches expected value")
			}
		}
		if !subtest.failed {
			t.Logf("    Got expected results. Subtest passes")
		}
	}
}

func TestAACAuthGetSnippetsForUser(t *testing.T) {
	aacAuth := newAACAuth(t)

	subtests := []testAACAuth{}
	subtests = append(subtests, testAACAuth{expectedIsError: true, subtestname: "No User", expectedError: auth.ErrUserNotSpecified})
	subtests = append(subtests, testAACAuth{expectedIsError: true, subtestname: "Fake User", userIdentity: "Fake User", expectedError: auth.ErrServiceNotSuccessful})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "Jonathan Holmes", userIdentity: "CN=Holmes Jonathan,OU=People,OU=Bedrock,OU=Six 3 Systems,O=U.S. Government,C=US"})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "Tester 10", userIdentity: "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us", expectedSnippets: []string{"dissem_countries allowed (USA)", "f_accms disallow ()", "f_atom_energy allowed ()", "f_clearance allowed (ts,s,c,u)", "f_macs disallow (tide,bir,watchdog)", "f_missions allowed ()", "f_oc_org allowed (dia)", "f_regions allowed ()", "f_sar_id allowed ()", "f_sci_ctrls disallow (el_eu,el_nk,rsv,hcs_o,el,kdk_kand,kdk,kdk_blfh,kdk_idit)", "dctc_odrive", "dctc_odrive_g1", "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus", "cusou_s_governmentouchimeraoudaeoupeoplecntesttester10"}})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "Uppercase Tester 10", userIdentity: "CN=Test Tester10,OU=People,OU=DAE,OU=Chimera,O=U.S. Government,C=US", expectedSnippets: []string{"dissem_countries allowed (USA)", "f_accms disallow ()", "f_atom_energy allowed ()", "f_clearance allowed (ts,s,c,u)", "f_macs disallow (tide,bir,watchdog)", "f_missions allowed ()", "f_oc_org allowed (dia)", "f_regions allowed ()", "f_sar_id allowed ()", "f_sci_ctrls disallow (el_eu,el_nk,rsv,hcs_o,el,kdk_kand,kdk,kdk_blfh,kdk_idit)", "dctc_odrive", "dctc_odrive_g1", "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus", "cusou_s_governmentouchimeraoudaeoupeoplecntesttester10"}})

	for testIdx, subtest := range subtests {
		t.Logf("Subtest %d: %s", testIdx, subtest.subtestname)
		snippets, err := aacAuth.GetSnippetsForUser(subtest.userIdentity)
		// If expecting an error but didn't get one
		if subtest.expectedIsError && err == nil {
			subtest.failed = true
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
			subtest.failed = true
			t.Logf("[x] Expected error (%s) but got error (%s)", subtest.expectedError.Error(), err.Error())
			t.Fail()
			continue
		}
		// If not expecting an error, but got one
		if !subtest.expectedIsError && err != nil {
			subtest.failed = true
			t.Logf("[x] Got an error %s", err.Error())
			t.Fail()
			continue
		}

		// If the snippets isn't what we expected
		if len(subtest.expectedSnippets) != 0 {
			// If the length of snippets is empty
			if snippets == nil {
				subtest.failed = true
				t.Logf("[x] No snippets were returned")
				t.Fail()
				continue
			}
			actualSnippets := snippets.String()
			for _, expected := range subtest.expectedSnippets {
				if !strings.Contains(actualSnippets, expected) {
					subtest.failed = true
					t.Logf("[x] Expected snippets to contain %s but got %s", subtest.expectedSnippets, actualSnippets)
					t.Fail()
					continue
				}
			}
		}
		if !subtest.failed {
			t.Logf("    Got expected results. Subtest passes")
		}
	}
}

func TestAACAuthInjectPermissionsIntoACM(t *testing.T) {

	aacAuth := newAACAuth(t)
	acmUnclassEveryone := `{"classif":"U","share":null,"version":"2.1.0"}`
	acmUnclassODrive := `{"classif":"U","share":{"projects":{"dctc":{"disp_nm":"DCTC","groups":["ODrive"]}}},"version":"2.1.0"}`
	acmUnclassODriveMod := `{"classif":"U","share":{"projects":{"dctc":{"disp_nm":"dctc","groups":["ODrive"]}}},"version":"2.1.0"}`
	acmUnclassODriveT1T2 := `{"classif":"U","share":{"projects":{"dctc":{"disp_nm":"dctc","groups":["ODrive"]}},"users":["cnt1","cnt2"]},"version":"2.1.0"}`
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
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "shared + no permissions unmodified", permissions: nopermissions, acm: acmUnclassODrive, expectedModifiedAcm: acmUnclassODriveMod})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "shared + permissions changes", permissions: permissionsT1T2, acm: acmUnclassODrive, expectedModifiedAcm: acmUnclassODriveT1T2})
	subtests = append(subtests, testAACAuth{expectedIsError: false, subtestname: "public + permissions changes", permissions: permissionsT1T2, acm: acmUnclassEveryone, expectedModifiedAcm: acmUnclassT1T2})

	for testIdx, subtest := range subtests {
		t.Logf("Subtest %d: %s", testIdx, subtest.subtestname)
		modifiedAcm, err := aacAuth.InjectPermissionsIntoACM(subtest.permissions, subtest.acm)
		// If expecting an error but didn't get one
		if subtest.expectedIsError && err == nil {
			subtest.failed = true
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
			subtest.failed = true
			t.Logf("[x] Expected error (%s) but got error (%s)", subtest.expectedError.Error(), err.Error())
			t.Fail()
			continue
		}
		// If not expecting an error, but got one
		if !subtest.expectedIsError && err != nil {
			subtest.failed = true
			t.Logf("[x] Got an error %s", err.Error())
			t.Fail()
			continue
		}
		// If the result isn't what we expect
		if subtest.expectedModifiedAcm != modifiedAcm {
			subtest.failed = true
			t.Logf("[x] Expected %s but got %s for injecting permissions", subtest.expectedModifiedAcm, modifiedAcm)
			t.Fail()
			continue
		}
		if !subtest.failed {
			t.Logf("    Got expected results. Subtest passes")
		}
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
	idDefault := "cn=NonExistent But Will Give Default,OU=People"
	idTester10 := "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	idTester01 := "cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"

	subtests := []testAACAuth{}
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "No User or ACM expect error", expectedError: auth.ErrACMNotSpecified})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "No User expect error", expectedError: auth.ErrUserNotSpecified, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "User with Invalid ACM expect error", userIdentity: idFake, expectedError: auth.ErrACMNotValid, acm: `{"x":"123"}`})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Fake User (-DIAS) for unclass expect error", userIdentity: idFake, expectedError: auth.ErrServiceNotSuccessful, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Nonexistent but gives default people (-DIAS) for unclass expect error", userIdentity: idDefault, expectedError: auth.ErrServiceNotSuccessful, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Jonathan Holmes (+DIAS) for unclass", userIdentity: idJon, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "David Yantz (+DIAS) for unclass expect error", userIdentity: idDave, expectedError: auth.ErrUserNotAuthorized, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Tester 10 (+DIAS) for unclass", userIdentity: idTester10, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Tester 01 (+DIAS) for unclass", userIdentity: idTester01, acm: acmUnclass})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Fake User (-DIAS) for secret expect error", userIdentity: idFake, expectedError: auth.ErrServiceNotSuccessful, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Nonexistent but gives default people (-DIAS) for secret expect error", userIdentity: idDefault, expectedError: auth.ErrServiceNotSuccessful, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Jonathan Holmes (+DIAS) for secret", userIdentity: idJon, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "David Yantz (+DIAS) for secret expect error", userIdentity: idDave, expectedError: auth.ErrUserNotAuthorized, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Tester 10 (+DIAS) for secret", userIdentity: idTester10, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Tester 01 (+DIAS) for secret expect error", userIdentity: idTester01, expectedError: auth.ErrUserNotAuthorized, acm: acmSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Fake User (-DIAS) for top secret expect error", userIdentity: idFake, expectedError: auth.ErrServiceNotSuccessful, acm: acmTopSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: true, expectedIsAuthorized: false, subtestname: "Nonexistent but gives default people (-DIAS) for top secret expect error", userIdentity: idDefault, expectedError: auth.ErrServiceNotSuccessful, acm: acmTopSecret})
	subtests = append(subtests, testAACAuth{expectedIsError: false, expectedIsAuthorized: true, subtestname: "Jonathan Holmes (+DIAS) for top secret", userIdentity: idJon, acm: acmTopSecret})
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
			subtest.failed = true
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
			subtest.failed = true
			t.Logf("[x] Expected error (%s) but got error (%s)", subtest.expectedError.Error(), err.Error())
			t.Fail()
			continue
		}
		// If not expecting an error, but got one
		if !subtest.expectedIsError && err != nil {
			subtest.failed = true
			t.Logf("[x] Got an error %s", err.Error())
			t.Fail()
			continue
		}
		// If the result isn't what we expect
		if subtest.expectedIsAuthorized != authorized {
			subtest.failed = true
			t.Logf("[x] Expected %t but got %t for authorized", subtest.expectedIsAuthorized, authorized)
			t.Fail()
			continue
		}
		if !subtest.failed {
			t.Logf("    Got expected results. Subtest passes")
		}
	}
}

// TestAACAuthGetGroupsFromSnippets tests both GetSnippetsForUser and GetGroupsFromSnippets
func TestAACAuthGetGroupsFromSnippets(t *testing.T) {
	aacAuth := newAACAuth(t)

	subtests := []testAACAuth{}
	subtests = append(subtests, testAACAuth{subtestname: "No User", userIdentity: "Fake User", expectedIsError: true, expectedIsAuthorized: false, expectedError: auth.ErrServiceNotSuccessful})
	subtests = append(subtests, testAACAuth{subtestname: "Nonexistent, default profile from dias simulator -- This should fail with dias:1.1.0 and newer. ok to update", userIdentity: "cn=NonExistent But Will Give Default,OU=People", expectedIsError: true, expectedIsAuthorized: false, expectedError: auth.ErrServiceNotSuccessful})
	subtests = append(subtests, testAACAuth{subtestname: "Tester 10", userIdentity: "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us", expectedIsError: false, expectedGroups: []string{"dctc_odrive", "dctc_odrive_g1", "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus", "cusou_s_governmentouchimeraoudaeoupeoplecntesttester10"}})
	subtests = append(subtests, testAACAuth{subtestname: "Tester 01", userIdentity: "cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us", expectedIsError: false, expectedGroups: []string{"dctc_odrive", "dctc_odrive_g2", "cntesttester01oupeopleoudaeouchimeraou_s_governmentcus", "cusou_s_governmentouchimeraoudaeoupeoplecntesttester01"}})

	for testIdx, subtest := range subtests {
		t.Logf("Subtest %d: %s", testIdx, subtest.subtestname)
		snippets, err := aacAuth.GetSnippetsForUser(subtest.userIdentity)
		// If expecting an error but didn't get one
		if subtest.expectedIsError && err == nil {
			subtest.failed = true
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
			subtest.failed = true
			t.Logf("[x] Expected error (%s) but got error (%s)", subtest.expectedError.Error(), err.Error())
			t.Fail()
			continue
		}
		// If not expecting an error, but got one
		if !subtest.expectedIsError && err != nil {
			subtest.failed = true
			t.Logf("[x] Got an error %s", err.Error())
			t.Fail()
			continue
		}
		groups := aacAuth.GetGroupsFromSnippets(snippets)
		for _, expectedGroup := range subtest.expectedGroups {
			expectedFoundInGroups := false
			for _, group := range groups {
				if group == expectedGroup {
					expectedFoundInGroups = true
					break
				}
			}
			if !expectedFoundInGroups {
				subtest.failed = true
				t.Logf("[x] Expected group '%s' was not found in snippets", expectedGroup)
				t.Fail()
			}
		}
		if !subtest.failed {
			t.Logf("    Got expected results. Subtest passes")
		}
	}
}

func TestAACAuthNormalizePermissionsFromACM(t *testing.T) {
	//t.Skip(`Skipping due to burdensome maintenance. New versions of AAC introduce HCS-P, and return macs and accms fields even when empty.`)
	aacAuth := newAACAuth(t)

	subtests := []testAACAuth{}
	subtests = append(subtests,
		testAACAuth{
			subtestname: "Tester10 owns with acm shared to tester01 and explicit tester10 CRUDS. Expect tester10 to get CRUDS and tester01 to have read only",
			creating:    false,
			owner:       "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
			acm:         `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"portion":"U//FOUO","share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`,
			permissions: []models.ODObjectPermission{
				models.ODObjectPermission{
					ID:          []byte{123, 45},
					Grantee:     "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					AllowCreate: true,
					AllowRead:   true,
					AllowUpdate: true,
					AllowDelete: true,
					AllowShare:  true,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester10"),
						UserDistinguishedName: models.ToNullString("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"),
					},
				},
			},
			expectedModifiedAcm: `{"accms":[],"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":["cntesttester01oupeopleoudaeouchimeraou_s_governmentcus","cntesttester10oupeopleoudaeouchimeraou_s_governmentcus"],"macs":[],"portion":"U//FOUO","share":{"projects":null,"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us","cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`,
			expectedModifiedPermissions: []models.ODObjectPermission{
				models.ODObjectPermission{
					Grantee:     "cntesttester01oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					AllowCreate: false,
					AllowRead:   true,
					AllowUpdate: false,
					AllowDelete: false,
					AllowShare:  false,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester01oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester01"),
						UserDistinguishedName: models.ToNullString("cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester01"),
					},
				},
				models.ODObjectPermission{
					Grantee:     "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					AllowCreate: true,
					AllowRead:   true,
					AllowUpdate: true,
					AllowDelete: true,
					AllowShare:  true,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester10"),
						UserDistinguishedName: models.ToNullString("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"),
					},
				},
			},
		},
		testAACAuth{
			subtestname: "Tester10 owns with acm shared to everyone and explicit tester10 CRUDS. Expect tester10 to get CUDS and everyone to have read only",
			creating:    false,
			owner:       "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
			acm:         `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"portion":"U//FOUO","version":"2.1.0"}`,
			permissions: []models.ODObjectPermission{
				models.ODObjectPermission{
					ID:          []byte{123, 45},
					Grantee:     "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					AllowCreate: true,
					AllowRead:   true,
					AllowUpdate: true,
					AllowDelete: true,
					AllowShare:  true,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester10"),
						UserDistinguishedName: models.ToNullString("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"),
					},
				},
			},
			expectedModifiedAcm: `{"accms":[],"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":[],"macs":[],"portion":"U//FOUO","version":"2.1.0"}`,
			expectedModifiedPermissions: []models.ODObjectPermission{
				models.ODObjectPermission{
					Grantee:     "_everyone",
					AcmShare:    `{"projects":{"":{"disp_nm":"","groups":["-Everyone"]}}}`,
					AllowCreate: false,
					AllowRead:   true,
					AllowUpdate: false,
					AllowDelete: false,
					AllowShare:  false,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:        "_everyone",
						DisplayName:    models.ToNullString("-Everyone"),
						GroupName:      models.ToNullString("-Everyone"),
						ResourceString: models.ToNullString("group/-Everyone/-Everyone"),
					},
				},
				// existing cruds permission gets deleted since it picks up that everyone grants the read
				models.ODObjectPermission{
					Grantee:     "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					IsDeleted:   true,
					AllowCreate: true,
					AllowRead:   true,
					AllowUpdate: true,
					AllowDelete: true,
					AllowShare:  true,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester10"),
						UserDistinguishedName: models.ToNullString("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"),
					},
				},
				// replacement permission for tester10
				models.ODObjectPermission{
					Grantee:     "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					AllowCreate: true,
					AllowRead:   false,
					AllowUpdate: true,
					AllowDelete: true,
					AllowShare:  true,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester10"),
						UserDistinguishedName: models.ToNullString("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"),
					},
				},
			},
		},
		testAACAuth{
			subtestname:         "Tester10 owns with acm shared to odrive group, no initial permissions declared. Expect odrive to get R, and tester10 to get CRUDS",
			creating:            false,
			owner:               "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
			acm:                 `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"portion":"U//FOUO","share":{"projects":{"dctc":{"disp_nm":"DCTC","groups":["ODrive"]}}},"version":"2.1.0"}`,
			expectedModifiedAcm: `{"accms":[],"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":["cntesttester10oupeopleoudaeouchimeraou_s_governmentcus","dctc_odrive"],"macs":[],"portion":"U//FOUO","share":{"projects":{"dctc":{"disp_nm":"dctc","groups":["odrive"]}},"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`,
			expectedModifiedPermissions: []models.ODObjectPermission{
				models.ODObjectPermission{
					Grantee:     "dctc_odrive",
					AcmShare:    `{"projects":{"dctc":{"disp_nm":"dctc","groups":["odrive"]}}}`,
					AllowCreate: false,
					AllowRead:   true,
					AllowUpdate: false,
					AllowDelete: false,
					AllowShare:  false,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:            "dctc_odrive",
						ProjectName:        models.ToNullString("dctc"),
						ProjectDisplayName: models.ToNullString("dctc"),
						DisplayName:        models.ToNullString("dctc odrive"),
						GroupName:          models.ToNullString("odrive"),
						ResourceString:     models.ToNullString("group/dctc/dctc/odrive/dctc odrive"),
					},
				},
				models.ODObjectPermission{
					Grantee:     "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					AllowCreate: true,
					AllowRead:   true,
					AllowUpdate: true,
					AllowDelete: true,
					AllowShare:  true,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester10"),
						UserDistinguishedName: models.ToNullString("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"),
					},
				},
			},
		},
		testAACAuth{
			subtestname: "Tester10 owns with acm private to self. Permissions grant to Tester01 create and read. Expect modified share to include Tester01",
			creating:    false,
			owner:       "user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
			acm:         `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"portion":"U//FOUO","share":{"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`,
			permissions: []models.ODObjectPermission{
				models.ODObjectPermission{
					Grantee:     "cntesttester01oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					AllowCreate: true,
					AllowRead:   true,
					AllowUpdate: false,
					AllowDelete: false,
					AllowShare:  false,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester01oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester01"),
						UserDistinguishedName: models.ToNullString("cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester01"),
					},
				},
			},
			expectedModifiedAcm: `{"accms":[],"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":["cntesttester10oupeopleoudaeouchimeraou_s_governmentcus","cntesttester01oupeopleoudaeouchimeraou_s_governmentcus"],"macs":[],"portion":"U//FOUO","share":{"projects":null,"users":["cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us","cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`,
			expectedModifiedPermissions: []models.ODObjectPermission{
				// originally in acm share
				models.ODObjectPermission{
					Grantee:     "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					AllowCreate: false,
					AllowRead:   true,
					AllowUpdate: false,
					AllowDelete: false,
					AllowShare:  false,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester10"),
						UserDistinguishedName: models.ToNullString("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"),
					},
				},
				// owner cruds
				models.ODObjectPermission{
					Grantee:     "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					AllowCreate: true,
					AllowRead:   true,
					AllowUpdate: true,
					AllowDelete: true,
					AllowShare:  true,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester10"),
						UserDistinguishedName: models.ToNullString("cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10"),
					},
				},
				// permissions created from initial acm share
				models.ODObjectPermission{
					Grantee:     "cntesttester01oupeopleoudaeouchimeraou_s_governmentcus",
					AcmShare:    fmt.Sprintf(`{"users":["%s"]}`, "cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
					AllowCreate: true,
					AllowRead:   true,
					AllowUpdate: false,
					AllowDelete: false,
					AllowShare:  false,
					AcmGrantee: models.ODAcmGrantee{
						Grantee:               "cntesttester01oupeopleoudaeouchimeraou_s_governmentcus",
						DisplayName:           models.ToNullString("test tester01"),
						UserDistinguishedName: models.ToNullString("cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"),
						ResourceString:        models.ToNullString("user/cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester01"),
					},
				},
			},
		},
	)

	for testIdx, subtest := range subtests {
		t.Logf("Subtest %d: %s", testIdx, subtest.subtestname)
		modifiedPermissions, modifiedAcm, err := aacAuth.NormalizePermissionsFromACM(subtest.owner, subtest.permissions, subtest.acm, subtest.creating)
		// If expecting an error but didn't get one
		if subtest.expectedIsError && err == nil {
			subtest.failed = true
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
			subtest.failed = true
			t.Logf("[x] Expected error (%s) but got error (%s)", subtest.expectedError.Error(), err.Error())
			t.Fail()
			continue
		}
		// If not expecting an error, but got one
		if !subtest.expectedIsError && err != nil {
			subtest.failed = true
			t.Logf("[x] Got an error %s", err.Error())
			t.Fail()
			continue
		}
		// If the result isn't what we expect
		if subtest.expectedModifiedAcm != modifiedAcm {
			subtest.failed = true
			t.Logf("[x] Expected acm of %s but got %s for normalizing permissions from acm", subtest.expectedModifiedAcm, modifiedAcm)
			t.Fail()
			continue
		}
		for _, modifiedPermission := range modifiedPermissions {
			modifiedPermissionInExpected := false
			for _, expectedModifiedPermission := range subtest.expectedModifiedPermissions {
				if modifiedPermission.IsDeleted == expectedModifiedPermission.IsDeleted {
					if strings.ToLower(modifiedPermission.String()) == strings.ToLower(expectedModifiedPermission.String()) {
						modifiedPermissionInExpected = true
					}
				}
			}
			if !modifiedPermissionInExpected {
				subtest.failed = true
				t.Logf("[x] Permission '%s' was not expected", modifiedPermission.String())
				t.Logf("    Deleted? %t", modifiedPermission.IsDeleted)
				t.Fail()
			}
		}
		for _, expectedModifiedPermission := range subtest.expectedModifiedPermissions {
			expectedFoundInModified := false
			for _, modifiedPermission := range modifiedPermissions {
				if modifiedPermission.IsDeleted == expectedModifiedPermission.IsDeleted {
					if strings.ToLower(modifiedPermission.String()) == strings.ToLower(expectedModifiedPermission.String()) {
						expectedFoundInModified = true
					}
				}
			}
			if !expectedFoundInModified {
				subtest.failed = true
				t.Logf("[x] Permission '%s' was not found in modified list", expectedModifiedPermission.String())
				t.Logf("    Deleted? %t", expectedModifiedPermission.IsDeleted)
				t.Fail()
			}
		}
		if !subtest.failed {
			t.Logf("    Got expected results. Subtest passes")
		}
	}
}

// Don't fail yet if we can't get an AAC - we need this to stall until it is available
func newAACAuthRaw() (*auth.AACAuth, error) {
	// AAC trust, client public & private key
	trustPath := filepath.Join("..", "defaultcerts", "client-aac", "trust", "client.trust.pem")
	certPath := filepath.Join("..", "defaultcerts", "client-aac", "id", "client.cert.pem")
	keyPath := filepath.Join("..", "defaultcerts", "client-aac", "id", "client.key.pem")
	serverCN := "twl-server-generic2"
	aacClient, err := aac.GetAACClient(getAACAddress(), getAACPort(), trustPath, certPath, keyPath, serverCN)
	if err != nil {
		return nil, err
	}
	aacAuth := auth.AACAuth{Logger: config.RootLogger, Service: aacClient}
	return &aacAuth, err
}

func stallForAvailability() int {
	// Don't stall on short tests
	if testing.Short() {
		return 0
	}

	// Do this on every try to check the server
	retryFunc := func() int {
		log.Printf("try connection: %s:%d", getAACAddress(), getAACPort())
		aacAuth, err := newAACAuthRaw()
		if err != nil {
			log.Printf("aac not ready: %v", err)
			return -11
		}
		// Just check that Jon comes back without blowing up, as a test that aac is ready.  Simply getting back aac isn't enough to ensure that we stop getting errors.
		_, err = aacAuth.GetSnippetsForUser("CN=Holmes Jonathan,OU=People,OU=Bedrock,OU=Six 3 Systems,O=U.S. Government,C=US")
		if err != nil {
			log.Printf("aac failing to get snippets for user, probably not ready: %v", err)
			return -10
		}
		return 0
	}

	// Try every few seconds
	tck := time.NewTicker(10 * time.Second)
	defer tck.Stop()

	// Give up after a while.  We need enough time to cover from when containers are brought up to when they should pass
	timeout := time.After(5 * time.Minute)

	// Attempt to check the server.  Quit if we pass timeout
	for {
		select {
		case <-tck.C:
			code := retryFunc()
			if code == 0 {
				return 0
			}
		case <-timeout:
			return -12
		}
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	stallForAvailability()
	os.Exit(m.Run())
}
