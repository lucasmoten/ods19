package server_test

import (
	"fmt"
	"strconv"
	"testing"

	"decipher.com/object-drive-server/util/testhelpers"
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

	acms := [...]string{"{\"version\":\"2.1.0\",\"classif\":\"U\",\"owner_prod\":[\"USA\"],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[\"NF\",\"FISA\"],\"non_ic\":[],\"rel_to\":[],\"declass_ex\":\"14X1-HUM\",\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"U//NF/FISA\",\"banner\":\"UNCLASSIFIED//NOFORN/FISA\",\"dissem_countries\":[\"USA\"],\"accms\":[],\"macs\":[{\"coi\":\"DEA\",\"disp_nm\":\"DEA\"}],\"oc_attribs\":[{}],\"f_clearance\":[\"u\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_atom_energy\":[],\"f_macs\":[\"dea\"],\"disp_only\":\"\"}", testhelpers.ValidACMTopSecretSITK, testhelpers.ValidACMUnclassified}

	// Create object as testperson10 with ACM that is TS
	for idx, acm := range acms {
		createdFolder, err := makeFolderWithACMViaJSON("TestACM "+strconv.Itoa(idx), acm, clientid1)
		if err != nil {
			t.Logf("Error making folder %d: %v", idx, err)
			t.FailNow()
		}
		if len(createdFolder.ID) == 0 {
			t.Logf("Object created has no ID")
			t.FailNow()
		}
	}
}
