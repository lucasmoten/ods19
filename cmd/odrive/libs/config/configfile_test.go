package config_test

import (
	"io/ioutil"
	"os"
	"testing"

	"decipher.com/object-drive-server/cmd/odrive/libs/config"
	"gopkg.in/yaml.v2"
)

func TestMain(m *testing.M) {
	m.Run()
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
