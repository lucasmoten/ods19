package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
)

func (h AppServer) listUsers(w http.ResponseWriter, r *http.Request, caller Caller) {

	w.Header().Set("Content-Type", "application/json")
	var users []models.ODUser
	users, err := h.DAO.GetUsers()
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Unable to get user list")
	}
	usersSerializable := mapping.MapODUsersToUsers(&users)
	converted, err := json.MarshalIndent(usersSerializable, "", "  ")
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Unable to get user list")
	}
	fmt.Fprintf(w, "%s", converted)
}
