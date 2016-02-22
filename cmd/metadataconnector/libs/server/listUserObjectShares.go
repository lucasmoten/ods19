package server

import (
	"encoding/json"
	"log"
	"net/http"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
)

func (h AppServer) listUserObjectShares(w http.ResponseWriter, r *http.Request, caller Caller) {
	result, err := h.DAO.GetObjectsSharedToMe(caller.DistinguishedName, "", 0, 20)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "GetObjectsSharedToMe query failed")
	}
	if r.Header.Get("Content-Type") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		returnValue := mapping.MapODObjectResultsetToObjectResultset(&result)
		data, err := json.MarshalIndent(returnValue, "", "  ")
		if err != nil {
			log.Printf("Error marshalling json data:%v", err)
		}
		w.Write(data)
	} else {
		// Removed HTML printing
		h.sendErrorResponse(w, 500, err, "listUserObjectShares requires content type application/json")
	}
}
