package server

import (
	"strings"

	"bitbucket.di2e.net/dime/object-drive-server/auth"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models/acm"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// GetUserGroupsAndSnippets fetches snippets and builds an array of groups for the user.
func (h AppServer) GetUserGroupsAndSnippets(ctx context.Context) ([]string, *acm.ODriveRawSnippetFields, error) {
	var groups []string
	logger := LoggerFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	var err error

	aacAuth := auth.NewAACAuth(logger, h.AAC)
	logger.Debug("getting snippets from context")
	snippetFields, ok := SnippetsFromContext(ctx)
	// From local profiles
	if !ok {
		logger.Debug("snippets were not found on context")
		// TODO(cm): should we perform this check outside of this function?
		PeerSignifier := config.GetEnvOrDefault(config.OD_PEER_SIGNIFIER, "P2P")
		if strings.ToLower(caller.UserDistinguishedName) == strings.ToLower(PeerSignifier) {
			// no snippets
			logger.Debug("caller is a peer, no snippets needed", zap.String("peersignifier", PeerSignifier))
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
