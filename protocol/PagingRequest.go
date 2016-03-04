package protocol

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/url"
	"strconv"
)

// PagingRequest supports a request constrained to a given page number and size
type PagingRequest struct {
	// PageNumber is the requested page number for this request
	PageNumber int `json:"pageNumber"`
	// PageSize is the requested page size for this request
	PageSize int `json:"pageSize"`
	// ParentID if provided lets us list the children
	ParentID string `json:"parentId"`
}

// NewPagingRequestFromQueryParams creates a new PagingRequest from the following URL params:
// pageNumber, pageSize, and parentId. Params are case-sensitive.
func NewPagingRequestFromURLValues(vals url.Values) (PagingRequest, error) {
	var pr PagingRequest
	pageNumberString := vals.Get("pageNumber")
	if pageNumberString == "" {
		return pr, errors.New("Must provide pageNumber to create PagingRequest from URL params.")
	}
	pageSizeString := vals.Get("pageSize")
	if pageSizeString == "" {
		return pr, errors.New("Must provide pageSize to create PagingRequest from URL params.")
	}

	parsedPageNumber, err := strconv.Atoi(pageNumberString)
	if err != nil {
		return pr, errors.New("Invalid pageNumber provided.")
	}

	parsedPageSize, err := strconv.Atoi(pageSizeString)
	if err != nil {
		return pr, errors.New("Invalid pageSize provided.")
	}
	pr.PageNumber, pr.PageSize = parsedPageNumber, parsedPageSize

	// parentID not required, so setting empty string is OK.
	parentIDString := vals.Get("parentId")
	pr.ParentID = parentIDString

	return pr, nil
}

// NewPagingRequestFromJSONBody parses a PagingRequest from a JSON body.
func NewPagingRequestFromJSONBody(body io.ReadCloser) (PagingRequest, error) {
	var pr PagingRequest
	var err error
	if body == nil {
		return pr, errors.New("JSON body was nil.")
	}
	err = (json.NewDecoder(body)).Decode(&pr)
	if err != nil {
		if err != io.EOF {
			log.Printf("Error parsing paging information in json: %v", err)
			return pr, err
		}
		// EOF ok. Reassign defaults and reset the error
		pr.PageNumber = 1
		pr.PageSize = 20
		err = nil
	}

	return pr, nil
}
