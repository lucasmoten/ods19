package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"decipher.com/oduploader/cmd/metadataconnector/libs/mapping"
	"decipher.com/oduploader/protocol"
	"golang.org/x/net/context"
)

func (h AppServer) listUserObjectsShared(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}

	// Parse Request
	pagingRequest, err := parseListUserObjectsSharedRequest(r)
	if err != nil {
		h.sendErrorResponse(w, 400, err, "Error parsing request")
		return
	}

	// Fetch matching objects
	sharedObjectsResultSet, err := h.DAO.GetObjectsIHaveShared("", pagingRequest.PageNumber, pagingRequest.PageSize, caller.DistinguishedName)
	if err != nil {
		h.sendErrorResponse(w, 500, err, "GetObjectsIHaveShared query failed")
	}

	// Render Response
	apiResponse := mapping.MapODObjectResultsetToObjectResultset(&sharedObjectsResultSet)
	listUserObjectsSharedResponseAsJSON(w, r, caller, &apiResponse)
	return
}

func parseListUserObjectsSharedRequest(r *http.Request) (*protocol.PagingRequest, error) {
	var jsonPaging protocol.PagingRequest
	defaultPage := 1
	defaultPageSize := 20
	jsonPaging.PageNumber = defaultPage
	jsonPaging.PageSize = defaultPageSize
	var err error

	err = (json.NewDecoder(r.Body)).Decode(&jsonPaging)
	if err != nil {
		// If there is no body, it's an EOF. So report other errors
		if err != io.EOF {
			log.Printf("Error parsing paging information in json: %v", err)
			return &jsonPaging, err
		}
		// EOF ok. Reassign defaults and reset the error
		jsonPaging.PageNumber = defaultPage
		jsonPaging.PageSize = defaultPageSize
		err = nil
	}

	// Paging provided as querystring arguments
	sPageNumber := r.URL.Query().Get("PageNumber")
	sPageSize := r.URL.Query().Get("PageSize")
	pageNumber, errPageNumber := strconv.Atoi(sPageNumber)
	if errPageNumber == nil && pageNumber > 0 {
		jsonPaging.PageNumber = pageNumber
	}
	pageSize, errPageSize := strconv.Atoi(sPageSize)
	if errPageSize == nil && pageSize > 0 {
		jsonPaging.PageSize = pageSize
	}
	if jsonPaging.PageNumber <= 0 {
		jsonPaging.PageNumber = defaultPage
	}
	if jsonPaging.PageSize <= 0 {
		jsonPaging.PageSize = defaultPageSize
	}

	return &jsonPaging, err
}

func listUserObjectsSharedResponseAsJSON(
	w http.ResponseWriter,
	r *http.Request,
	caller Caller,
	response *protocol.ObjectResultset,
) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		log.Printf("Error marshalling response as json: %s", err.Error())
		return
	}
	w.Write(jsonData)
}
