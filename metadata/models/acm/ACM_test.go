package acm_test

import (
	"strings"
	"testing"

	"decipher.com/oduploader/metadata/models/acm"
	"decipher.com/oduploader/util/testhelpers"
)

func TestUnmarshallKnownACMs(t *testing.T) {

	ParsedACMUnclassified, err := acm.NewACMFromRawACM(testhelpers.ValidACMUnclassified)
	if err != nil {
		t.Error(err)
		t.Failed()
	}
	if strings.Compare(ParsedACMUnclassified.OverallBanner, "UNCLASSIFIED") != 0 {
		t.Logf("Expected UNCLASSIFIED, got %s", ParsedACMUnclassified.OverallBanner)
		t.Failed()
	}

	ParsedACMUnclassifiedFOUO, err := acm.NewACMFromRawACM(testhelpers.ValidACMUnclassifiedFOUO)
	if err != nil {
		t.Error(err)
		t.Failed()
	}
	if strings.Compare(ParsedACMUnclassifiedFOUO.OverallBanner, "UNCLASSIFIED//FOUO") != 0 {
		t.Logf("Expected UNCLASSIFIED//FOUO, got %s", ParsedACMUnclassifiedFOUO.OverallBanner)
		t.Failed()
	}

}

func TestUnmarshallShortACMs(t *testing.T) {

	acms := []string{
		`{"classif":"U"}`,
		`{"classif":"S"}`,
		`{"version":"2.1.0","classif":"U","portion":"u","banner":"UNCLASSIFIED","dissem_countries":["USA"]}`,
		`{ "path": "", "classif":"TS", "sci_ctrls":[ "HCS", "SI-G", "TK" ], "dissem_ctrls":[ "OC" ], "dissem_countries":[ "USA" ], "oc_attribs":[ { "orgs":[ "dia" ] } ] }`,
		`{ "version":"2.1.0", "classif":"TS", "owner_prod":[], "atom_energy":[], "sar_id":[], "sci_ctrls":[ "HCS", "SI-G", "TK" ], "disponly_to":[ "" ], "dissem_ctrls":[ "OC" ], "non_ic":[], "rel_to":[], "fgi_open":[], "fgi_protect":[], "portion":"TS//HCS/SI-G/TK//OC", "banner":"TOP SECRET//HCS/SI-G/TK//ORCON", "dissem_countries":[ "USA" ], "accms":[], "macs":[], "oc_attribs":[ { "orgs":[ "dia" ], "missions":[], "regions":[] } ], "f_clearance":[ "ts" ], "f_sci_ctrls":[ "hcs", "si_g", "tk" ], "f_accms":[], "f_oc_org":[ "dia", "dni" ], "f_regions":[], "f_missions":[], "f_share":[], "f_atom_energy":[], "f_macs":[], "disp_only":"" }`,
		`{"version":"2.1.0","classif":"S","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":["NF"],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"S//NF","banner":"SECRET//NOFORN","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["s"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
		`{"version":"2.1.0","classif":"C","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"C","banner":"CONFIDENTIAL","dissem_countries":["USA"],"f_clearance":["c"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
		`{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":["FOUO"],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U//FOUO","banner":"UNCLASSIFIED//FOUO","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`,
	}

	for i, shortacm := range acms {
		parsedACM, err := acm.NewACMFromRawACM(shortacm)
		if err != nil {
			t.Logf("Error unmarshalling shortacm #%d: %v", i, err)
			t.Failed()
		}
		if len(parsedACM.Classif) == 0 {
			t.Logf("Parsed classif was empty for shortacm #%d", i)
			t.Failed()
		}
	}

}
