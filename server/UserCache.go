package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"bitbucket.di2e.net/dime/object-drive-server/auth"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models/acm"
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
	h.UsersLruCache.Set(caller.DistinguishedName, *user, time.Duration(config.GetEnvOrDefaultInt(config.OD_USERAOCACHE_LRU_TIME, 600))*time.Second)

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

// CheckUserAOCache examines the cache state for a user. If none exists, then
// it will be built. If a cache does exist, but its snippet definition differs
// as identified by comparing against hash values, then it will be rebuilt, if
// and only if no other process is already rebuilding it, unless the time since
// call to rebuild and the current date of the cache is older then a specified
// duration (for now hardcoded as 2 minutes)
func (h AppServer) CheckUserAOCache(ctx context.Context) error {
	timein := time.Now().UTC()
	logger := LoggerFromContext(ctx)
	caller, _ := CallerFromContext(ctx)
	dao := DAOFromContext(ctx)
	snippets, hasSnippets := SnippetsFromContext(ctx)
	if !hasSnippets || snippets == nil {
		return nil
	}
	// Normalizes, serializes, and hashes...
	logger.Debug("calculating snippet hash")
	snippetHash := calculateSnippetHash(snippets)
	user, _ := UserFromContext(ctx)
	user.Snippets = snippets
	rebuild := false
	built := false
	fetchfromdb := true
	var useraocache models.ODUserAOCache
	var err error

	if cacheItem := h.UserAOsLruCache.Get(caller.DistinguishedName); cacheItem != nil {
		if cacheItem.Expired() {
			logger.Debug("existing user ao cache from lru is expired, checking database")
			cacheItem.Release()
			fetchfromdb = true
		} else {
			logger.Debug("getting user ao cache from lru memory cache")
			useraocache = cacheItem.Value().(models.ODUserAOCache)
			fetchfromdb = false
		}
	}
	if fetchfromdb {
		logger.Debug("getting user ao cache from DB")
		useraocache, err = dao.GetUserAOCacheByDistinguishedName(user)
	}
	// If no user ao cache yet ..
	if err != nil {
		logger.Debug("no cache yet")
		if err == sql.ErrNoRows {
			logger.Info("user ao cache will be built because it does not exist", zap.String("dn", caller.DistinguishedName), zap.String("userdn", user.DistinguishedName))
			rebuild = true
			useraocache.UserID = user.ID
		} else {
			// Unrecoverable error
			logger.Warn("unable to get user ao cache", zap.Error(err), zap.String("dn", caller.DistinguishedName))
			return err
		}
	} else {
		logger.Debug("cache exists, checking hash value")
		// If hash isn't the same...
		if useraocache.SHA256Hash != snippetHash {
			if !useraocache.IsCaching {
				logger.Info("user ao cache will be built because the hash of the snippets changed", zap.String("dn", caller.DistinguishedName), zap.String("userdn", user.DistinguishedName), zap.String("oldhash", useraocache.SHA256Hash), zap.String("newhash", snippetHash))
				rebuild = true
			} else {
				// another process is doing this work..
				logger.Info("cache is already being rebuilt for change to hash", zap.String("dn", caller.DistinguishedName))
			}
		} else {
			logger.Debug("hash is same, checking if caching")
			// hash is the same, see if caching
			if useraocache.IsCaching {
				logger.Debug("user ao is caching")
				// if caching and older than configured timeout
				if time.Since(useraocache.CacheDate.Time).Seconds() > float64(config.GetEnvOrDefaultInt(config.OD_USERAOCACHE_TIMEOUT, 40)) {
					// something may be wrong (or else we're going to create a race condition)
					msg := fmt.Sprintf("cache rebuild for user took longer than %d seconds. Rebuilding", config.GetEnvOrDefaultInt(config.OD_USERAOCACHE_TIMEOUT, 40))
					logger.Warn(msg, zap.String("dn", caller.DistinguishedName))
					rebuild = true
				} else {
					msg := fmt.Sprintf("cache being rebuilt for same hash but has not exceeded %d seconds", config.GetEnvOrDefaultInt(config.OD_USERAOCACHE_TIMEOUT, 40))
					logger.Info(msg, zap.String("dn", caller.DistinguishedName))
				}
			} else {
				logger.Debug("not caching, no rebuild needed")
			}
		}
	}

	if rebuild {
		logger.Debug("rebuilding user cache")
		// Init
		useraocache.UserID = user.ID
		useraocache.CacheDate.Time = time.Now().UTC()
		useraocache.CacheDate.Valid = true
		useraocache.IsCaching = true
		useraocache.SHA256Hash = snippetHash
		// Random delay and recheck
		if isUserAOCacheBeingBuilt(dao, user, useraocache) {
			logger.Info("peer cache rebuild happening in parallel. skipping")
		} else {
			// Save the cache definition
			logger.Info("saving user cache placeholder")
			if err := dao.SetUserAOCacheByDistinguishedName(&useraocache, user); err != nil {
				logger.Warn("error saving user ao cache", zap.Error(err))
				return err
			}
			// With user attributes, add grantees (and resulting keys) for missing project/groups
			// The dao for this is responsible for saving the grantee, as well as acm value via
			// database trigger if not yet created, so that it can be appropriately associated to
			// user if referenced in an acm later
			aacAuth := auth.NewAACAuth(logger, h.AAC)
			logger.Info("getting user attributes to base cache on")
			userAttributes, err := aacAuth.GetAttributesForUser(caller.UserDistinguishedName)
			if err != nil {
				logger.Warn("error retrieving user attributes", zap.Error(err))
				return err
			}
			logger.Info("adding necessary groups")
			for _, diasProject := range userAttributes.DIASUserGroups.Projects {
				if len(strings.TrimSpace(diasProject.Name)) == 0 {
					logger.Warn("dias project name is empty, skipping")
					continue
				}
				for _, groupName := range diasProject.Groups {
					if len(strings.TrimSpace(groupName)) == 0 {
						logger.Warn("dias project group name is empty, skipping")
						continue
					}
					resourceName := fmt.Sprintf("group/%s/%s", diasProject.Name, groupName)
					acmGrantee := models.NewODAcmGranteeFromResourceName(resourceName)
					if _, err := h.RootDAO.CreateAcmGrantee(acmGrantee); err != nil {
						logger.Warn("error saving new acmgrantee", zap.Error(err), zap.String("acmGrantee", acmGrantee.ResourceName()), zap.String("grantee", acmGrantee.Grantee), zap.String("diasProject", diasProject.Name), zap.String("diasProject.Group", groupName))
						continue
					}
				}
			}
			// Build links from user to acm
			runasync := false
			// Short circuited mini builds are front loaded based upon request characteristics
			if method, ok := RequestMethodFromContext(ctx); ok && method == "GET" {
				if uri, ok := RequestURIFromContext(ctx); ok && h.Routes.Objects.RX.MatchString(uri) {
					done := make(chan bool, 1)
					logger.Info("building initial user cache for userroot")
					dao.RebuildUserACMCache(&useraocache, user, done, "userroot")
					built = true
					runasync = true
				}
			}
			// Full rebuild
			if runasync {
				logger.Info("building full user cache asynchronously")
				built = true
				go func() {
					done := make(chan bool)
					timeout := time.After(time.Duration(config.GetEnvOrDefaultInt(config.OD_USERAOCACHE_TIMEOUT, 40)) * time.Second)
					go dao.RebuildUserACMCache(&useraocache, user, done, "")

					for {
						select {
						case <-timeout:
							logger.Warn("checkuseraocache call to rebuilduseracmcache has timed out")
							return
						case <-done:
							logger.Info("asynchronous cache build completed")
							h.UserAOsLruCache.Set(caller.DistinguishedName, useraocache, time.Duration(config.GetEnvOrDefaultInt(config.OD_USERAOCACHE_LRU_TIME, 600))*time.Second)
							return
						}
					}
				}()
			} else {
				logger.Info("building full user cache synchronously")
				done := make(chan bool, 1)
				dao.RebuildUserACMCache(&useraocache, user, done, "")
				logger.Info("synchronous cache build completed")
				built = true
			}
		}
	}

	logger.Debug("saving useraocache to lru memory cache", zap.String("user", caller.DistinguishedName), zap.Bool("iscaching", useraocache.IsCaching))
	timetocache := time.Duration(config.GetEnvOrDefaultInt(config.OD_USERAOCACHE_LRU_TIME, 600)) * time.Second
	if useraocache.IsCaching {
		timetocache = time.Duration(config.GetEnvOrDefaultInt(config.OD_USERAOCACHE_TIMEOUT, 40)) * time.Second
	}
	h.UserAOsLruCache.Set(caller.DistinguishedName, useraocache, timetocache)

	if rebuild {
		for !built && isUserAOCacheBeingBuilt(dao, user, useraocache) {
			if time.Since(timein).Seconds() > float64(config.GetEnvOrDefaultInt(config.OD_USERAOCACHE_TIMEOUT, 40)) {
				return fmt.Errorf("checkuseraocache waiting for cache being built until ready has timed out")
			}
		}
	}

	return nil
}

func isUserAOCacheBeingBuilt(dao dao.DAO, user models.ODUser, desired models.ODUserAOCache) bool {
	// Random delay from 50-200 ms to permit parallel operations from same request to commence
	// Checked in DB because peer nodes may be servicing request due to nginx RR and need to centralize
	minimumDelay := 50
	maximumDelay := 200
	randomDelay := rand.New(rand.NewSource(time.Now().UTC().UnixNano())).Intn(maximumDelay-minimumDelay) + minimumDelay
	time.Sleep(time.Duration(randomDelay) * time.Millisecond)
	// Now check state
	actual, err := dao.GetUserAOCacheByDistinguishedName(user)
	// if errors (typically sql.ErrNoRows, then not being built)
	if err != nil {
		return false
	}
	// caching status
	if !actual.IsCaching {
		return false
	}
	// overdue
	if time.Since(actual.CacheDate.Time).Seconds() > float64(config.GetEnvOrDefaultInt(config.OD_USERAOCACHE_TIMEOUT, 40)) {
		return false
	}
	// hash state
	if actual.SHA256Hash != desired.SHA256Hash {
		return false
	}
	return true
}

func calculateSnippetHash(snippets *acm.ODriveRawSnippetFields) string {

	// Build sorted field name list
	fieldNames := make([]string, len(snippets.Snippets))
	for fi, field := range snippets.Snippets {
		fieldNames[fi] = field.FieldName
	}
	sort.Strings(fieldNames)

	// Build up serialized snippet representation
	var flattenedSnippets string
	for fi, fieldName := range fieldNames {
		if fi > 0 {
			flattenedSnippets += ";"
		}
		flattenedSnippets += fieldName
		for _, snippetField := range snippets.Snippets {
			if snippetField.FieldName == fieldName {
				flattenedSnippets += fmt.Sprintf(":%s=", snippetField.Treatment)
				values := make([]string, len(snippetField.Values))
				for vi, value := range snippetField.Values {
					values[vi] = value
				}
				sort.Strings(values)
				for vi, value := range values {
					if vi > 0 {
						flattenedSnippets += ","
					}
					flattenedSnippets += value
				}
			}
		}
	}

	// Hash it
	shabytes := sha256.Sum256([]byte(flattenedSnippets))
	snippetHash := fmt.Sprintf("%x", shabytes)
	return snippetHash
}
