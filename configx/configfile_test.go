package config_test

import (
	"io/ioutil"
	"os"
	"testing"

	"decipher.com/object-drive-server/configx"

	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestParseWhitelistFromConfigFile(t *testing.T) {

	contents := readAllOrFail("testfixtures/testconf.yml", t)
	var conf config.ConfigFile
	err := yaml.Unmarshal(contents, &conf)
	if err != nil {
		t.Errorf("Could not unmarshal yaml config file: %v\n", err)
	}

	if len(conf.Whitelisted) != 3 {
		t.Fail()
	}
	if conf.Whitelisted[0] != "first" {
		t.Fail()
	}

}

func readAllOrFail(path string, t *testing.T) []byte {
	f, err := os.Open(path)
	if err != nil {
		t.Fail()
	}
	contents, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fail()
	}
	return contents
}

func TestNormalization(t *testing.T) {

	checklist := make(map[string]string)

	checklist["/C=US/O=U.S. Government/OU=twl-server-generic2/OU=DIA/OU=DAE/CN=twl-server-generic2"] = "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	checklist["CN=twl-server-generic2, OU=DAE, OU=DIA, OU=twl-server-generic2, O=U.S. Government, C=US"] = "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"

	for startingValue, expected := range checklist {
		actual := config.GetNormalizedDistinguishedName(startingValue)
		if actual != expected {
			t.Logf("Normalized %s to %s. Expected %s", startingValue, actual, expected)
			t.Fail()
		}
	}
}
