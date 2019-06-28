package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// LoadYAMLConfig constructs an AppConfiguration struct from a YAML file.
func LoadYAMLConfig(path string) (AppConfiguration, error) {
	var conf AppConfiguration
	f, err := os.Open(path)
	// DIMEODS-1262 - ensure file closed if not nil
	if f != nil {
		defer f.Close()
	}
	if err != nil {
		return conf, err
	}

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal(contents, &conf)
	if err != nil {
		return conf, err
	}

	return conf, nil
}
