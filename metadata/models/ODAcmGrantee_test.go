package models_test

import (
	"testing"

	"github.com/deciphernow/object-drive-server/metadata/models"
)

func TestNewODAcmGranteeFromResourceName(t *testing.T) {

	resourceStringsInput := []string{
		"user/cn=test tester 10",
		"group/-Everyone",
		"group/Some other group",
		"group/-Everyone/-Everyone",
		"group/dctc/DCTC/ODrive/DCTC ODrive",
		"user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester10",
		"user/cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester01",
		"user/cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester02",
		"user/cn=test tester03,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester03",
		"user/cn=test tester04,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester04",
		"user/cn=test tester05,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester05",
		"user/cn=test tester06,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester06",
		"user/cn=test tester07,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester07",
		"user/cn=test tester08,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester08",
		"user/cn=test tester09,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us/test tester09",
		"group/dctc/DCTC/ODrive_G1/DCTC ODrive_G1",
		"group/dctc/DCTC/ODrive_G2/DCTC ODrive_G2",
	}
	resourceStringsExpected := []string{
		"user/cn=test tester 10",
		"group/-everyone",
		"group/some other group",
		"group/-everyone",
		"group/dctc/dctc/odrive",
		"user/cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		"user/cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		"user/cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		"user/cn=test tester03,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		"user/cn=test tester04,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		"user/cn=test tester05,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		"user/cn=test tester06,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		"user/cn=test tester07,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		"user/cn=test tester08,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		"user/cn=test tester09,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us",
		"group/dctc/dctc/odrive_g1",
		"group/dctc/dctc/odrive_g2",
	}
	grantees := []string{
		"cntesttester10",
		"_everyone",
		"someothergroup",
		"_everyone",
		"dctc_odrive",
		"cntesttester10oupeopleoudaeouchimeraou_s_governmentcus",
		"cntesttester01oupeopleoudaeouchimeraou_s_governmentcus",
		"cntesttester02oupeopleoudaeouchimeraou_s_governmentcus",
		"cntesttester03oupeopleoudaeouchimeraou_s_governmentcus",
		"cntesttester04oupeopleoudaeouchimeraou_s_governmentcus",
		"cntesttester05oupeopleoudaeouchimeraou_s_governmentcus",
		"cntesttester06oupeopleoudaeouchimeraou_s_governmentcus",
		"cntesttester07oupeopleoudaeouchimeraou_s_governmentcus",
		"cntesttester08oupeopleoudaeouchimeraou_s_governmentcus",
		"cntesttester09oupeopleoudaeouchimeraou_s_governmentcus",
		"dctc_odrive_g1",
		"dctc_odrive_g2",
	}
	groupNames := []string{
		"",
		"-everyone",
		"some other group",
		"-everyone",
		"odrive",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"odrive_g1",
		"odrive_g2",
	}

	for idx, resourceString := range resourceStringsInput {
		t.Logf("%3d) %s", idx, resourceString)
		t.Logf("-> Creating ODAcmGrantee from resource name")
		odAcmGrantee := models.NewODAcmGranteeFromResourceName(resourceString)
		t.Logf("-> Checking Grantee against expected value %s", grantees[idx])
		if odAcmGrantee.Grantee != grantees[idx] {
			t.Logf("[x]Grantee for %d did not match expected value %s. Got %s", idx, grantees[idx], odAcmGrantee.Grantee)
			t.Fail()
		}
		expectedGroup := groupNames[idx]
		t.Logf("-> Checking Group Name against expected value %s", expectedGroup)
		if odAcmGrantee.GroupName.String != expectedGroup {
			t.Logf("[x]Group Name for %d did not match expected value %s. Got %s", idx, expectedGroup, odAcmGrantee.GroupName.String)
			t.Fail()
		}
		expectedRS := resourceStringsExpected[idx]
		t.Logf("-> Checking Resource Name against expected value %s", expectedRS)
		if odAcmGrantee.ResourceName() != expectedRS {
			t.Logf("[x]Resource Name for %d did not match expected value %s. Got %s", idx, expectedRS, odAcmGrantee.ResourceName())
			t.Fail()
		}
	}

}
