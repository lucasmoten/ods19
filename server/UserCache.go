package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/uber-go/zap"

	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/metadata/models"
	"golang.org/x/net/context"
)

// FetchUser examines the context on the request, and retrieves the matching
// user either from cache, or from the database, creating the record as appropriate
func (h AppServer) FetchUser(ctx context.Context) (*models.ODUser, error) {

	caller, ok := CallerFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("caller was not set on context")
	}
	dao := DAOFromContext(ctx)

	if cacheItem := h.UsersLruCache.Get(caller.DistinguishedName); cacheItem != nil {
		user := cacheItem.Value().(models.ODUser)
		return &user, nil
	}

	// Not found in cache, look up from database
	user, err := getOrCreateUser(dao, caller)
	if err != nil {
		return nil, err
	}

	// Basic validation on the user object to make sure modifiedBy is set by trigger
	if len(user.ModifiedBy) == 0 {
		jsonData, err := json.MarshalIndent(user, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("Error marshalling user as JSON")
		}
		LoggerFromContext(ctx).Warn("user does not have modified by set", zap.String("json", string(jsonData)))
		return nil, fmt.Errorf("User created when fetching user is not in expected state")
	}
	// Finally, add this user to this server's cache
	h.UsersLruCache.Set(caller.DistinguishedName, *user, time.Minute*10)

	return user, nil
}

// getOrCreateUser attempts to retrieve an existing user by their DN. If no user is found, exactly
// two attempts are made to create the user.
func getOrCreateUser(dao dao.DAO, caller Caller) (*models.ODUser, error) {

	query := models.ODUser{DistinguishedName: caller.DistinguishedName}
	existing, err := dao.GetUserByDistinguishedName(query)
	if err == nil {
		return &existing, nil
	}

	if err == sql.ErrNoRows {

		simpleRetry := func(data models.ODUser) (*models.ODUser, error) {
			time.Sleep(500 * time.Millisecond)
			if existing2, err := dao.GetUserByDistinguishedName(data); err == nil {
				return &existing2, nil
			}
			created, err := dao.CreateUser(data)
			if err != nil {
				return nil, err
			}
			return &created, nil
		}

		// Not yet in database, we need to add them
		newUser := models.ODUser{
			DistinguishedName: caller.DistinguishedName,
			DisplayName:       models.ToNullString(caller.CommonName),
			CreatedBy:         caller.DistinguishedName,
		}
		created, err := dao.CreateUser(newUser)
		if err != nil {
			return simpleRetry(newUser)
		}
		return &created, nil
	}

	return nil, fmt.Errorf("error communicating with database to get user")
}
