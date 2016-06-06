package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/odrive/libs/mapping"
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
	snippetFields, err := h.FetchUserSnippets(ctx)
	if err != nil {
		return NewAppError(502, errors.New("Error retrieving user permissions."), err.Error())
	} else {
		// Fake the groups from the snippets as additional elements in the users array
		for _, rawSnippetField := range snippetFields.Snippets {
			if strings.Compare(rawSnippetField.FieldName, "f_share") == 0 {
				for _, shareValue := range rawSnippetField.Values {
					// Exclude the share to self which is a collapsed DN format forward and reversed.
					// Samples are:
					//      cntesttester10oupeopleoudaeouchimeraou_s_governmentcus
					//      cusou_s_governmentouchimeraoudaeoupeoplecntesttester10
					if !strings.Contains(shareValue, "cusou") && !strings.Contains(shareValue, "governmentcus") {
						var groupName models.ODUser
						groupName.DistinguishedName = shareValue
						groupName.DisplayName.String = "Group:" + shareValue
						groupName.DisplayName.Valid = true
						usersAndGroups = append(usersAndGroups, groupName)
					}
				}
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

	// Write output
	w.Header().Set("Content-Type", "application/json")
	usersSerializable := mapping.MapODUsersToUsers(&odusers)
	converted, err := json.MarshalIndent(usersSerializable, "", "  ")
	if err != nil {
		return NewAppError(500, err, "Unable to get user list")
	}
	w.Write(converted)
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
