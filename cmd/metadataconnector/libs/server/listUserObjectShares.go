package server

import (
	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/metadata/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
		h.listUserObjectSharesAsHTML(w, r, caller, &result)
	}
}

func (h AppServer) listUserObjectSharesAsHTML(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *models.ODObjectResultset,
) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "listUserObjectShares", caller.DistinguishedName)
	fmt.Fprintf(w, "<h2>SharedTo:%s<h2>", caller.CommonName)
	h.listObjectsResponseAsHTMLTable(w, response)
	fmt.Fprintf(w, pageTemplateEnd)
}
