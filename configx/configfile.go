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
	if err != nil {
		return conf, err
	}
	defer f.Close()

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
