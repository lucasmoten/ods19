package protocol

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

// PagingRequest supports a request constrained to a given page number and size
type PagingRequest struct {
	// PageNumber is the requested page number for this request
	PageNumber int `json:"pageNumber"`
	// PageSize is the requested page size for this request
	PageSize int `json:"pageSize"`
	// ObjectID if provided provides a focus for paging, often the ParentID
	ObjectID string `json:"objectId"`
}

// NewPagingRequestFromQueryParams creates a new PagingRequest from the following URL params:
// pageNumber, pageSize, and parentId. Params are case-sensitive.
func NewPagingRequestFromURLValues(vals url.Values) (PagingRequest, error) {

	pagingRequest := PagingRequest{}

	// Paging provided as querystring arguments
	pagingRequest.PageNumber = GetQueryParamAsPositiveInt(vals, []string{"PageNumber", "pageNumber"}, 1)
	pagingRequest.PageSize = GetQueryParamAsPositiveInt(vals, []string{"PageSize", "pageSize"}, 20)

	// parentID not required, so setting empty string is OK.
	parentIDString := vals.Get("parentId")
	pagingRequest.ObjectID = parentIDString

	return pagingRequest, nil
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

// NewPagingRequestWithObjectID parses a paging request provided in either the
// query string arguments of a GET, or the application/json body of a POST
// along with a key object identifier based upon provided regular expression
// to match on
func NewPagingRequestWithObjectID(r *http.Request, pathRegex *regexp.Regexp, isObjectIDRequired bool) (*PagingRequest, error) {
	var pagingRequest *PagingRequest

	// Paging information...
	if r.Method == "GET" {
		vals := r.URL.Query()
		pr, _ := NewPagingRequestFromURLValues(vals)
		pagingRequest = &pr
	} else if r.Method == "POST" {
		if r.Header.Get("Content-Type") == "application/json" {
			pr, err := NewPagingRequestFromJSONBody(r.Body)
			if err != nil {
				return nil, errors.New("Error parsing request from message body")
			}
			pagingRequest = &pr
		} else {
			return nil, errors.New("Unsupported content-type to parse paging information")
		}
	} else {
		return nil, errors.New("Unsupported HTTP Method")
	}

	// Object identifier...
	if pathRegex != nil {
		uri := r.URL.Path
		matchIndexes := pathRegex.FindStringSubmatchIndex(uri)
		if len(matchIndexes) != 0 {
			if len(matchIndexes) > 3 {
				pagingRequest.ObjectID = uri[matchIndexes[2]:matchIndexes[3]]
				_, err := hex.DecodeString(pagingRequest.ObjectID)
				if err != nil {
					return pagingRequest, errors.New("Object Identifier in Request URI is not a hex string")
				}
			}
		}
	}
	// Validation check
	if isObjectIDRequired && len(pagingRequest.ObjectID) != 32 {
		return nil, errors.New("Object Identifier not found in Request URI")
	}

	// All ready and no errors
	return pagingRequest, nil
}

func GetQueryParamAsPositiveInt(vals url.Values, names []string, defaultValue int) int {
	rv := 0
	found := false
	for _, n := range names {
		if len(n) > 0 {
			s := vals.Get(n)
			if len(s) > 0 {
				v, err := strconv.Atoi(s)
				if err == nil && v > 0 {
					found = true
					rv = v
					break
				}
			}
		}
	}
	if !found {
		rv = defaultValue
	}
	return rv
}
