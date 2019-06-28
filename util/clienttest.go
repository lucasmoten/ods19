package util

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func GetSchemeAuthority() string {
	return fmt.Sprintf("https://%s:%s", GetEnvWithDefault("OD_EXTERNAL_HOST", "proxier"), GetEnvWithDefault("OD_EXTERNAL_PORT", "8080"))
}

func GetClientMountPoint() string {
	version := "0.0"
	changelogFilePath := os.Getenv("GOPATH") + "/src/bitbucket.di2e.net/dime/object-drive-server/changelog.md"
	f, _ := os.Open(changelogFilePath)
	if f != nil {
		defer f.Close()
	}
	reader := bufio.NewReader(f)
	re := regexp.MustCompile("## Release (?P<version>[v\\.0-9]*) ")
	for {
		line, _ := reader.ReadString('\n')
		if re.MatchString(line) {
			groups := GetRegexCaptureGroups(line, re)
			version = groups["version"]
			version = strings.Replace(version, "v", "", -1)
			versionParts := strings.Split(version, ".")
			version = fmt.Sprintf("%s.%s", versionParts[0], versionParts[1])
			break
		}
	}
	out := fmt.Sprintf("%s/services/object-drive/%s", GetSchemeAuthority(), version)
	return out
}
