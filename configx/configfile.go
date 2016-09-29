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
	AppConfiguration
}

// LoadYAMLConfig constructs a (legacy) ConfigFile struct as well as an
// AppConfiguration from a YAML file.
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
