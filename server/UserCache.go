package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/metadata/models"
	"golang.org/x/net/context"
)

// FetchUser examines the context on the request, and retrieves the matching
// user either from cache, or from the database, creating the record as appropriate
func (h AppServer) FetchUser(ctx context.Context) (*models.ODUser, error) {

	// Caller info from context
	caller, ok := CallerFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("Could not get caller when fetching user")
	}
	dao := DAOFromContext(ctx)

	var user *models.ODUser

	// First check if exists in the cache
	cacheItem := h.UsersLruCache.Get(caller.DistinguishedName)
	if cacheItem != nil {
		userObject := cacheItem.Value().(models.ODUser)
		user = &userObject
	} else {
		// Not found in cache, look up from database
		var userRequested models.ODUser
		userRequested.DistinguishedName = caller.DistinguishedName
		userRetrievedFromDB, err := dao.GetUserByDistinguishedName(userRequested)
		if err != nil {
			if err == sql.ErrNoRows {
				// Not yet in database, we need to add them
				userRequested.DistinguishedName = caller.DistinguishedName
				userRequested.DisplayName.String = caller.CommonName
				userRequested.DisplayName.Valid = true
				userRequested.CreatedBy = caller.DistinguishedName
				userRetrievedFromDB, err = dao.CreateUser(userRequested)
				if err != nil {
					LoggerFromContext(ctx).Error(
						"user does not exist",
						zap.String("err", err.Error()),
					)
					return nil, fmt.Errorf("Error access resource when creating user")
				}
			} else {
				// Some other database error
				LoggerFromContext(ctx).Error(
					"error getting user from database",
					zap.String("err", err.Error()),
				)
				return nil, fmt.Errorf("Error communicating with database to get user.")
			}
		}
		// Basic validation on the user object to make sure modifiedBy is set
		// (when a record created in db, modifiedBy is assigned by a trigger copying createdBy)
		if len(userRetrievedFromDB.ModifiedBy) == 0 {
			jsonData, err := json.MarshalIndent(user, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("Error marshalling user as JSON")
			}
			LoggerFromContext(ctx).Warn("user does not have modified by set", zap.String("json", string(jsonData)))
			return nil, fmt.Errorf("User created when fetching user is not in expected state")
		}
		// Finally, add this user to this server's cache
		h.UsersLruCache.Set(caller.DistinguishedName, userRetrievedFromDB, time.Minute*10)
		user = &userRetrievedFromDB
	}
	return user, nil
}
