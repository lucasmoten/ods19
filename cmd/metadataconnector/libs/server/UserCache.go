package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"decipher.com/object-drive-server/metadata/models"
	"golang.org/x/net/context"
)

// UserCache is a simple in memory cache to hold user info looked up from
// database to reduce database calls on nearly every request
type UserCache struct {
	lock *sync.RWMutex
	data map[string]*models.ODUser
}

// NewUserCache instantiates a pointer to a UserCache.
func NewUserCache() *UserCache {
	l := &sync.RWMutex{}
	data := make(map[string]*models.ODUser)
	return &UserCache{lock: l, data: data}
}

// Get retrieves a user from the cache
func (uc *UserCache) Get(key string) (*models.ODUser, bool) {
	uc.lock.RLock()
	defer uc.lock.RUnlock()
	if uc.data == nil {
		uc.data = make(map[string]*models.ODUser)
	}
	d, ok := uc.data[key]
	return d, ok
}

// Set assigns a user to the cache
func (uc *UserCache) Set(key string, d *models.ODUser) {
	uc.lock.Lock()
	defer uc.lock.Unlock()
	if uc.data == nil {
		uc.data = make(map[string]*models.ODUser)
	}
	uc.data[key] = d
}

// Delete removes an entry from the cache
func (uc *UserCache) Delete(key string) {
	uc.lock.Lock()
	defer uc.lock.Unlock()
	if uc.data != nil {
		delete(uc.data, key)
	}
}

// Clear removes all entries from the cache
func (uc *UserCache) Clear() {
	uc.lock.Lock()
	defer uc.lock.Unlock()
	uc.data = make(map[string]*models.ODUser)
}

// FetchUser examines the context on the request, and retrieves the matching
// user either from cache, or from the database, creating the record as appropriate
func (h AppServer) FetchUser(ctx context.Context) (*models.ODUser, error) {

	// Caller info from context
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("Could not get caller when fetching user")
	}

	// First check if exists in the cache
	user, ok := h.Users.Get(caller.DistinguishedName)
	if !ok {
		// Not found in cache, look up from database
		log.Printf("Looking up user %s in database", caller.DistinguishedName)
		var userRequested models.ODUser
		userRequested.DistinguishedName = caller.DistinguishedName
		userRetrievedFromDB, err := h.DAO.GetUserByDistinguishedName(userRequested)
		if err != nil {
			if err == sql.ErrNoRows {
				// Not yet in database, we need to add them
				userRequested.DistinguishedName = caller.DistinguishedName
				userRequested.DisplayName.String = caller.CommonName
				userRequested.DisplayName.Valid = true
				userRequested.CreatedBy = caller.DistinguishedName
				userRetrievedFromDB, err = h.DAO.CreateUser(userRequested)
				if err != nil {
					log.Printf("%s does not exist in database. Error creating: %s", caller.DistinguishedName, err.Error())
					return nil, fmt.Errorf("Error access resource when creating user")
				}
			} else {
				// Some other database error
				log.Printf("Error getting user from database: %s", err.Error())
				return nil, fmt.Errorf("Error communicating with database to get user.")
			}
		} else {
			log.Printf("User %s retrieved from database", userRetrievedFromDB.DistinguishedName)
		}
		// Basic validation on the user object to make sure modifiedBy is set
		// (when a record created in db, modifiedBy is assigned by a trigger copying createdBy)
		if len(userRetrievedFromDB.ModifiedBy) == 0 {
			log.Println("User does not have modified by set!")
			jsonData, err := json.MarshalIndent(user, "", "  ")
			if err != nil {
				log.Println("Error marshalling as JSON!")
				return nil, fmt.Errorf("Error marshalling user as JSON")
			}
			jsonified := string(jsonData)
			log.Println(jsonified)
			return nil, fmt.Errorf("User created when fetching user is not in expected state")
		}
		// Finally, add this user to this server's cache
		h.Users.Set(caller.DistinguishedName, &userRetrievedFromDB)
		user = &userRetrievedFromDB
	}
	return user, nil
}
