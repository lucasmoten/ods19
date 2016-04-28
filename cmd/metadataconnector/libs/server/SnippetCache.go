package server

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/metadata/models/acm"
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
	snippets, ok := h.Snippets.Get(user.DistinguishedName)
	if !ok {
		// Not found in cache, look up from aac
		log.Printf("Looking up snippets for user %s from aac", user.DistinguishedName)

		// Call AAC to get Snippets
		snippetType := "odrive-raw"
		snippetResponse, err := h.AAC.GetSnippets(user.DistinguishedName, "pki_dias", snippetType)
		if err != nil {
			log.Printf("Error calling AAC.GetSnippets: %s", err.Error())
			return nil, err
		}
		if !snippetResponse.Success {
			messages := "Failed to successfully retrieve snippets for user. Messages = "
			for _, message := range snippetResponse.Messages {
				messages += message
			}
			log.Printf("Calling AAC.GetSnippets was not successful: %s", messages)
			return nil, fmt.Errorf(messages)
		}

		// Convert to Snippet Fields
		odriveRawSnippetFields, err := acm.NewODriveRawSnippetFieldsFromSnippetResponse(snippetResponse.Snippets)
		if err != nil {
			log.Printf("Error converting snippets to snippet fields: %s", err.Error())
			return nil, err
		}

		// Add this user snippet to this server's cache
		h.Snippets.Set(user.DistinguishedName, &odriveRawSnippetFields)
		snippets = &odriveRawSnippetFields
	}
	return snippets, nil
}