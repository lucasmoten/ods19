package server

import (
	"strings"

	"github.com/deciphernow/object-drive-server/auth"
	"github.com/deciphernow/object-drive-server/ciphertext"
	"github.com/deciphernow/object-drive-server/metadata/models/acm"
	"golang.org/x/net/context"
)

// GetUserGroupsAndSnippets fetches snippets and builds an array of groups for the user.
func (h AppServer) GetUserGroupsAndSnippets(ctx context.Context) ([]string, *acm.ODriveRawSnippetFields, error) {
	var groups []string
	logger := LoggerFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	var err error

	aacAuth := auth.NewAACAuth(logger, h.AAC)
	snippetFields, ok := SnippetsFromContext(ctx)
	// From local profiles
	if !ok {
		// TODO(cm): should we perform this check outside of this function?
		if strings.ToLower(caller.UserDistinguishedName) == strings.ToLower(ciphertext.PeerSignifier) {
			// no snippets
			ok = true
		}
	}
	// From AAC
	if !ok {
		snippetFields, err = aacAuth.GetSnippetsForUser(caller.UserDistinguishedName)
		if err != nil {
			return nil, nil, err
		}
	}
	groups = aacAuth.GetGroupsFromSnippets(snippetFields)
	return groups, snippetFields, nil
}
