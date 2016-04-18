package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/metadata/models"
)

func (h AppServer) listUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	var users []models.ODUser
	users, err := h.DAO.GetUsers()
	if err != nil {
		sendErrorResponse(&w, 500, err, "Unable to get user list")
		return
	}
	usersSerializable := mapping.MapODUsersToUsers(&users)
	converted, err := json.MarshalIndent(usersSerializable, "", "  ")
	if err != nil {
		sendErrorResponse(&w, 500, err, "Unable to get user list")
		return
	}
	fmt.Fprintf(w, "%s", converted)
	countOKResponse()
}
