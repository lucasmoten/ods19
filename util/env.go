package util

import "os"

func GetEnvWithDefault(name string, def string) string {
	val := os.Getenv(name)
	if val == "" {
		return def
	}
	return val
}
