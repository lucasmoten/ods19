package config_test

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.di2e.net/dime/object-drive-server/config"

	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestParseWhitelistFromConfigFile(t *testing.T) {

	contents := readAllOrFail(t, "testfixtures/testconf.yml")
	var conf config.AppConfiguration
	err := yaml.Unmarshal(contents, &conf)
	if err != nil {
		t.Errorf("Could not unmarshal yaml config file: %v\n", err)
	}

	if len(conf.ServerSettings.ACLImpersonationWhitelist) != 3 {
		t.Fail()
	}
	if conf.ServerSettings.ACLImpersonationWhitelist[0] != "first" {
		t.Fail()
	}

}
func TestParseAppConfigurationFromConfigFile(t *testing.T) {
	// os.Getenv() save out value to test cascade
	reset1 := unsetReset("OD_DB_PORT")
	defer reset1()

	reset2 := unsetReset("OD_EVENT_ZK_ADDRS")
	defer reset2()

	reset3 := unsetReset("OD_EVENT_TOPIC")
	defer reset3()

	contents := readAllOrFail(t, "testfixtures/complete.yml")
	var conf config.AppConfiguration
	err := yaml.Unmarshal(contents, &conf)
	if err != nil {
		t.Errorf("Could not unmarshal yaml config file: %v\n", err)
	}

	if conf.DatabaseConnection.Driver != "mysql" {
		t.Errorf("expected mysql, got: %v", conf.DatabaseConnection.Driver)
	}
	if conf.AACSettings.CAPath != "foo" {
		t.Errorf("expected foo, got: %v", conf.AACSettings.CAPath)
	}

	if conf.DatabaseConnection.Port != "9999" {
		t.Errorf("expected 9999, got %v", conf.DatabaseConnection.Port)
	}

	if len(conf.EventQueue.ZKAddrs) != 2 {
		t.Errorf("expected zk_addrs string slice of len 2, got: %v", conf.EventQueue.ZKAddrs)
	}
	if conf.EventQueue.Topic != "odrive-event" {
		t.Errorf("expected odrive-event, got: %v", conf.EventQueue.Topic)
	}
	if conf.ServerSettings.ACLImpersonationWhitelist[0] != "foo" {

		t.Errorf("expected whitelist entry foo but got: %s", conf.ServerSettings.ACLImpersonationWhitelist[0])
	}

}

func readAllOrFail(t *testing.T, path string) []byte {
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

func unsetReset(env string) func() {
	original := os.Getenv(env)
	os.Setenv(env, "")
	return func() {
		os.Setenv(env, original)
	}
}
