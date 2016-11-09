package server

import (
	"net/http"
	"sort"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/metadata/models"
)

func (h AppServer) listUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Retreive the users
	var users []models.ODUser
	dao := DAOFromContext(ctx)

	users, err := dao.GetUsers()
	if err != nil {
		return NewAppError(500, err, "Unable to get user list")
	}
	// Alter the returned users to
	//      Remove any that look like groups
	//      Change display name to prefix with user
	var usersAndGroups userContainer
	for _, oduser := range users {
		if strings.Contains(oduser.DistinguishedName, "=") {
			oduser.DisplayName.String = "User:" + oduser.DisplayName.String
			usersAndGroups = append(usersAndGroups, oduser)
		}
	}

	// Get snippets for user, which will have group membership
	if groups, ok := GroupsFromContext(ctx); ok {
		for _, group := range groups {
			if !strings.Contains(group, "cusou") && !strings.Contains(group, "governmentcus") {
				var groupName models.ODUser
				groupName.DistinguishedName = group
				groupName.DisplayName.String = "Group:" + group
				groupName.DisplayName.Valid = true
				usersAndGroups = append(usersAndGroups, groupName)
			}
		}
	}

	// Sort
	sort.Sort(usersAndGroups)

	// Recast back to type
	odusers := make([]models.ODUser, len(usersAndGroups))
	for i := range usersAndGroups {
		odusers[i] = usersAndGroups[i]
	}

	apiResponse := mapping.MapODUsersToUsers(&odusers)
	jsonResponse(w, apiResponse)
	return nil
}

type userContainer []models.ODUser

func (slice userContainer) Len() int {
	return len(slice)
}
func (slice userContainer) Less(i, j int) bool {
	return slice[i].DisplayName.String < slice[j].DisplayName.String
}
func (slice userContainer) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
