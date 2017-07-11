package acm

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/uber-go/zap"
)

// ODriveUserAttributes is a structure to hold the dias groups returned from user attributes
type ODriveUserAttributes struct {
	UserDN         string         `json:"userDN"`
	DIASUserGroups diasUserGroups `json:"diasUserGroups"`
}
type diasUserGroups struct {
	Projects []diasProjects `json:"projects"`
}
type diasProjects struct {
	Name   string   `json:"projectName"`
	Groups []string `json:"groupNames"`
}

// NewODriveAttributesFromAttributeResponse takes the entire userattributes response from AAC and builds the user attributes struct
func NewODriveAttributesFromAttributeResponse(userattributes string) (ODriveUserAttributes, error) {
	var userAttributes ODriveUserAttributes
	var err error

	//The attributes may be escape-quoted. If so, resolve this
	unquotedAttributes := userattributes
	if strings.HasPrefix(userattributes, `{\`) {
		unquotedAttributes, err = strconv.Unquote(userattributes)
		if err != nil {
			logger.Error("acm attributes unquoting error", zap.Object("attributes", userattributes), zap.Object("err", err.Error()))
			return userAttributes, err
		}
	}

	jsonIOReader := bytes.NewBufferString(unquotedAttributes)
	err = (json.NewDecoder(jsonIOReader)).Decode(&userAttributes)
	if err != nil {
		logger.Error("acm attributes unparseable", zap.Object("attributes", unquotedAttributes))
		return userAttributes, err
	}

	return userAttributes, nil
}
