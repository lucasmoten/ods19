package server

import (
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"encoding/json"
	//"log"
	"net/http"
)

func (h AppServer) listUsers(w http.ResponseWriter, r *http.Request, caller Caller) {
	w.Header().Set("Content-Type", "application/json")
	var users []models.ODUser
	users, err := h.DAO.GetUsers()
	if err != nil {
		h.sendErrorResponse(w, 500, err, "Unable to get user list")
	}
	encoder := json.NewEncoder(w)
	usersEncodeable := mapping.MapODUsersToUsers(&users)
	encoder.Encode(usersEncodeable)
}
