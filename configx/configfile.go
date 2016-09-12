package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// ConfigFile is the object representation of Object Drive's main config file.
// Most of Object Drive is configured with environment variables, but not all
// options make sense for environment variables, particularly lists.
type ConfigFile struct {
	Whitelisted []string `yaml:"whitelist"`
}

// LoadYAMLConfig constructs a ConfigFile struct from a YAML file.
func LoadYAMLConfig(path string) (ConfigFile, error) {
	var conf ConfigFile

	f, err := os.Open(path)
	if err != nil {
		return conf, err
	}
	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return conf, err
	}
	err = yaml.Unmarshal(contents, &conf)
	return conf, nil
}
