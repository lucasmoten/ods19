package server_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"decipher.com/object-drive-server/server"
)

func TestAddObjectACMs(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	verboseOutput := testing.Verbose()
	clientid1 := 0

	if verboseOutput {
		t.Logf("(Verbose Mode) Using client id %d", clientid1)
		fmt.Println()
	}

	acmSkipReasons := make(map[int]string)
	acmSkipReasons[0] = "User does not have proper MAC access"
	acmSkipReasons[12] = "User does not belong to the correct Organization"
	acmSkipReasons[13] = "User does not belong to the correct Organization"

	var acms []string

	acms = append(acms, "{\"version\":\"2.1.0\",\"classif\":\"U\",\"owner_prod\":[\"USA\"],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[\"NF\",\"FISA\"],\"non_ic\":[],\"rel_to\":[],\"declass_ex\":\"14X1-HUM\",\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"U//NF/FISA\",\"banner\":\"UNCLASSIFIED//NOFORN/FISA\",\"dissem_countries\":[\"USA\"],\"accms\":[],\"macs\":[{\"coi\":\"DEA\",\"disp_nm\":\"DEA\"}],\"oc_attribs\":[{}],\"f_clearance\":[\"u\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_atom_energy\":[],\"f_macs\":[\"dea\"],\"disp_only\":\"\"}")
	acms = append(acms, server.ValidACMTopSecretSITK)
	acms = append(acms, server.ValidACMUnclassified)

	// Here are some more that should be valid
	acms = append(acms, `{"banner":"TOP SECRET//SI/TK","classif":"TS","dissem_countries":["USA"],"f_clearance":["ts"],"f_sci_ctrls":["si","tk"],"owner_prod":["USA"],"portion":"TS//SI/TK","sci_ctrls":["si","tk"],"version":"2.1.0"}`)
	acms = append(acms, `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"f_clearance":["u"],"owner_prod":["USA"],"portion":"U","version":"2.1.0"}`)
	acms = append(acms, `{"banner":"UNCLASSIFIED//FOUO","classif":"U","dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_clearance":["u"],"owner_prod":["USA"],"portion":"U//FOUO","version":"2.1.0"}`)
	acms = append(acms, `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED","classif":"U","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":[],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"missions":[],"orgs":[],"regions":[]}],"owner_prod":[],"portion":"U","rel_to":[],"sar_id":[],"sci_ctrls":[],"version":"2.1.0"}`)
	acms = append(acms, `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED","classif":"U","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":[],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"missions":[],"orgs":[],"regions":[]}],"owner_prod":["USA"],"portion":"U","rel_to":[],"sar_id":[],"sci_ctrls":[],"version":"2.1.0"}`)
	acms = append(acms, `{"atom_energy":[],"banner":"UNCLASSIFIED","classif":"U","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":[],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"non_ic":[],"owner_prod":[],"portion":"U","rel_to":[],"sar_id":[],"sci_ctrls":[],"version":"2.1.0"}`)
	acms = append(acms, `{"banner":"SECRET//NF","classif":"S","dissem_countries":["USA"],"dissem_ctrls":["nf"],"f_clearance":["s"],"owner_prod":["USA"],"portion":"S//NF","version":"2.1.0"}`)
	acms = append(acms, `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED//NOFORN/PROPIN","classif":"U","declass_ex":"40X1-HUM","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":["NF","PR"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{}],"owner_prod":["USA"],"portion":"U//NF/PR","rel_to":[],"sar_id":[],"sci_ctrls":[],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive"]}}},"version":"2.1.0"}`)
	acms = append(acms, `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED","classif":"TS","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":[],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"missions":[],"orgs":[],"regions":[]}],"owner_prod":[],"portion":"U","rel_to":[],"sar_id":[],"sci_ctrls":[],"version":"2.1.0"}`)
	acms = append(acms, `{"accms":[],"atom_energy":[],"banner":"SECRET//SI//ORCON/NOFORN","classif":"S","declass_ex":"56X1-HUM","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":["OC","NF"],"f_accms":[],"f_atom_energy":[],"f_clearance":["s"],"f_macs":[],"f_missions":[],"f_oc_org":["dia","dod_dia","dni"],"f_regions":[],"f_sci_ctrls":["si"],"f_share":[],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"orgs":["DIA","DOD_DIA"]}],"owner_prod":["USA"],"portion":"S//SI//OC/NF","rel_to":[],"sar_id":[],"sci_ctrls":["SI"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive"]}}},"version":"2.1.0"}`)
	acms = append(acms, `{"accms":[],"atom_energy":[],"banner":"TOP SECRET//TK//ORCON/NOFORN","classif":"TS","declass_ex":"62X1-HUM","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":["OC","NF"],"f_accms":[],"f_atom_energy":[],"f_clearance":["ts"],"f_macs":[],"f_missions":[],"f_oc_org":["dia","dod_dia","dni"],"f_regions":[],"f_sci_ctrls":["tk"],"f_share":[],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"orgs":["DIA","DOD_DIA"]}],"owner_prod":["USA"],"portion":"TS//TK//OC/NF","rel_to":[],"sar_id":[],"sci_ctrls":["TK"],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive"]}}},"version":"2.1.0"}`)
	acms = append(acms, `{"atom_energy":[],"banner":"SECRET","classif":"S","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":[],"f_accms":[],"f_atom_energy":[],"f_clearance":["s"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"non_ic":[],"owner_prod":[],"portion":"S","rel_to":[],"sar_id":[],"sci_ctrls":[],"share":{"projects":null,"users":null},"version":"2.1.0"}`)
	acms = append(acms, `{"banner":"UNCLASSIFIED","classif":"U","dissem_countries":["USA"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":[],"portion":"U","version":"2.1.0"}`)
	acms = append(acms, `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED","classif":"U","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":[],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"missions":[],"orgs":[],"regions":[]}],"owner_prod":[],"portion":"U","rel_to":[],"sar_id":[],"sci_ctrls":[],"version":"2.1.0"}`)
	acms = append(acms, `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED","classif":"U","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":[],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"missions":[],"orgs":[],"regions":[]}],"owner_prod":["USA"],"portion":"U","rel_to":[],"sar_id":[],"sci_ctrls":[],"version":"2.1.0"}`)
	acms = append(acms, `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED//NOFORN/PROPIN","classif":"U","declass_ex":"40X1-HUM","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":["NF","PR"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"missions":null,"orgs":null,"regions":null}],"owner_prod":["USA"],"portion":"U//NF/PR","rel_to":[],"sar_id":[],"sci_ctrls":[],"version":"2.1.0"}`)
	acms = append(acms, `{"atom_energy":[],"banner":"CONFIDENTIAL","classif":"C","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":[],"f_accms":[],"f_atom_energy":[],"f_clearance":["c"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"non_ic":[],"owner_prod":[],"portion":"C","rel_to":[],"sar_id":[],"sci_ctrls":[],"share":{"projects":null,"users":null},"version":"2.1.0"}`)

	// Create object as testperson10 with ACM that is TS
	for idx, acm := range acms {
		acmSkipReason := acmSkipReasons[idx]
		createdFolder, err := makeFolderWithACMViaJSON("TestACM "+strconv.Itoa(idx), acm, clientid1)
		if len(acmSkipReason) > 0 {
			if err == nil {
				t.Logf("Skipped test for %d is no longer reporting error.", idx)
			} else {
				if !strings.Contains(err.Error(), acmSkipReason) {
					t.Logf("Skipped test %d didnt fail for expected reason. Actual error was %s", idx, err.Error())
					t.Fail()
				}
			}
		} else {
			if err != nil {
				t.Logf("Error making folder for test %d: %v", idx, err.Error())
				t.Fail()
			} else if len(createdFolder.ID) == 0 {
				t.Logf("Object created for test %d has no ID", idx)
				t.Fail()
			}
		}
	}
}

func TestAddObjectACMWithShare(t *testing.T) {

	acm := `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED","classif":"U","disp_only":"","disponly_to":[""],"dissem_countries":[],"dissem_ctrls":[],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_regions":[],"f_sci_ctrls":[],"f_share":[],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"owner_prod":["USA"],"portion":"U","rel_to":[],"sar_id":[],"sci_ctrls":[],"share":{"projects":{"DCTC":{"disp_nm":"DCTC","groups":["ODrive","ODrive_G1","ODrive_G2"]}}},"version":"2.1.0"}`

	clientid1 := 0
	createdFolder, err := makeFolderWithACMViaJSON("TestACM With Share", acm, clientid1)
	if err != nil {
		t.Logf("Error making folder: %v", err)
		t.FailNow()
	}
	if len(createdFolder.ID) == 0 {
		t.Logf("Object created has no ID")
		t.FailNow()
	}

}
