package server

import (
	"errors"
	"fmt"
	"strings"

	"sync"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
	"decipher.com/object-drive-server/performance"
	"golang.org/x/net/context"
)

// SnippetCachce is a simple in memory cache to hold user snippet info obtained
// from AAC to reduce outbound calls to dependent service on list type requests
type SnippetCache struct {
	lock *sync.RWMutex
	data map[string]*acm.ODriveRawSnippetFields
}

// NewSnippetCache instantiates a pointer to a SnippetCache.
func NewSnippetCache() *SnippetCache {
	l := &sync.RWMutex{}
	data := make(map[string]*acm.ODriveRawSnippetFields)
	return &SnippetCache{lock: l, data: data}
}

// Get retrieves a user snippet from the cache
func (sc *SnippetCache) Get(key string) (*acm.ODriveRawSnippetFields, bool) {
	sc.lock.RLock()
	defer sc.lock.RUnlock()
	if sc.data == nil {
		sc.data = make(map[string]*acm.ODriveRawSnippetFields)
	}
	d, ok := sc.data[key]
	return d, ok
}

// Set assigns a user snippet to the cache
func (sc *SnippetCache) Set(key string, d *acm.ODriveRawSnippetFields) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	if sc.data == nil {
		sc.data = make(map[string]*acm.ODriveRawSnippetFields)
	}
	sc.data[key] = d
}

// Delete removes an entry from the cache
func (sc *SnippetCache) Delete(key string) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	if sc.data != nil {
		delete(sc.data, key)
	}
}

// Clear removes all entries from the cache
func (sc *SnippetCache) Clear() {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	sc.data = make(map[string]*acm.ODriveRawSnippetFields)
}

// FetchUserSnippets examines the context on the request, and retrieves the matching
// user either from cache, or from the database, creating the record as appropriate
func (h AppServer) FetchUserSnippets(ctx context.Context) (*acm.ODriveRawSnippetFields, error) {

	// TODO Should we provide an alternative method that takes a user? This is called from
	// http handler functions, and the first thing those functions do is grab User or Caller
	// off the context object.
	var cacheUserSnippets = false

	// Get user from context
	user, ok := UserFromContext(ctx)
	if !ok {
		caller, ok := CallerFromContext(ctx)
		if !ok {
			return nil, errors.New("Could not determine user")
		}
		user = models.ODUser{DistinguishedName: caller.DistinguishedName}
	}

	// First check if exists in the cache
	var snippets *acm.ODriveRawSnippetFields
	if cacheUserSnippets {
		snippets, ok = h.Snippets.Get(user.DistinguishedName)
	}
	if !ok || !cacheUserSnippets {
		// Performance instrumentation
		var beganAt = performance.BeganJob(int64(0))
		if h.Tracker != nil {
			beganAt = h.Tracker.BeginTime(performance.AACCounterGetSnippets)
		}

		LoggerFromContext(ctx).Info("look up snippets")

		// Verify we have a reference to AAC
		if h.AAC == nil {
			return nil, errors.New("AAC field is nil.")
		}

		// Call AAC to get Snippets
		snippetType := "odrive-raw"
		snippetResponse, err := h.AAC.GetSnippets(user.DistinguishedName, "pki_dias", snippetType)
		if err != nil {
			LoggerFromContext(ctx).Error(
				"error calling AAC.GetSnippets",
				zap.String("err", err.Error()),
			)
			return nil, err
		}
		if !snippetResponse.Success {
			messages := "Failed to successfully retrieve snippets for user. Messages = "
			for _, message := range snippetResponse.Messages {
				messages += message
			}
			LoggerFromContext(ctx).Error(
				"AAC.GetSnippets unsuccessful",
				zap.String("messages", messages),
			)
			return nil, fmt.Errorf(messages)
		}
		//LoggerFromContext(ctx).Warn("    SNIPPETS    ", zap.String("snippetResponse.Snippets", snippetResponse.Snippets))
		// Convert to Snippet Fields
		odriveRawSnippetFields, err := acm.NewODriveRawSnippetFieldsFromSnippetResponse(snippetResponse.Snippets)
		if err != nil {
			LoggerFromContext(ctx).Error(
				"error converting snippets to fields",
				zap.String("err", err.Error()),
			)
			return nil, err
		}

		// Performance tracking
		if h.Tracker != nil {
			h.Tracker.EndTime(
				performance.AACCounterGetSnippets,
				beganAt,
				performance.SizeJob(1),
			)
		}

		// Add this user snippet to this server's cache
		if cacheUserSnippets {
			h.Snippets.Set(user.DistinguishedName, &odriveRawSnippetFields)
		}
		snippets = &odriveRawSnippetFields
	}
	return snippets, nil
}

// GetUserGroupsAndSnippets fetches snippets and builds an array of groups for the user.
func (h AppServer) GetUserGroupsAndSnippets(ctx context.Context) ([]string, *acm.ODriveRawSnippetFields, error) {
	var groups []string

	snippetFields, ok := SnippetsFromContext(ctx)
	if !ok {
		var err error
		snippetFields, err = h.FetchUserSnippets(ctx)
		if err != nil {
			LoggerFromContext(ctx).Error(
				"error fetching user snippets",
				zap.String("err", err.Error()),
			)
			// return empty list
			return groups, snippetFields, err
		}
	}
	// Values of f_share are those the user gets as groups
	for _, rawSnippetField := range snippetFields.Snippets {
		if strings.Compare(rawSnippetField.FieldName, "f_share") == 0 {
			for _, shareValue := range rawSnippetField.Values {
				groups = append(groups, shareValue)
			}
		}
	}
	return groups, snippetFields, nil
}
