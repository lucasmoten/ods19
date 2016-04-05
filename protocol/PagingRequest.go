package protocol

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
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
	// FilterSettings is an array of fitler settings denoting field and conditional match expression to filter results
	FilterSettings []FilterSetting `json:"filterSettings"`
	// SortSettings is an array of sort settings denoting a field to sort on and direction
	SortSettings []SortSetting `json:"sortSettings"`
}

// newPagingRequestFromURLValues creates a new PagingRequest from the following URL params:
// pageNumber, pageSize, and parentId. Params are case-sensitive.
func newPagingRequestFromURLValues(vals url.Values) (PagingRequest, error) {

	pagingRequest := PagingRequest{}

	// Paging provided as querystring arguments
	pagingRequest.PageNumber = GetQueryParamAsPositiveInt(vals, []string{"PageNumber", "pageNumber"}, 1)
	pagingRequest.PageSize = GetQueryParamAsPositiveInt(vals, []string{"PageSize", "pageSize"}, 20)

	pagingRequest.FilterSettings = makeFilterSettingsFromQueryParam(vals)
	pagingRequest.SortSettings = makeSortSettingsFromQueryParam(vals)

	// parentID not required, so setting empty string is OK.
	parentIDString := vals.Get("parentId")
	if len(parentIDString) > 0 {
		// Assign it
		pagingRequest.ObjectID = parentIDString
		// Validate that it can be decoded
		_, err := hex.DecodeString(pagingRequest.ObjectID)
		if err != nil {
			return pagingRequest, errors.New("Object Identifier in Request URI is not a hex string")
		}
	}

	return pagingRequest, nil
}

// newPagingRequestFromJSONBody parses a PagingRequest from a JSON body.
func newPagingRequestFromJSONBody(body io.ReadCloser) (PagingRequest, error) {
	var pr PagingRequest
	var err error
	if body == nil {
		return pr, errors.New("JSON body was nil")
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

// NewPagingRequest parses a paging request provided in either the
// query string arguments of a GET, or the application/json body of a POST
// along with a key object identifier populated from capture groups
func NewPagingRequest(r *http.Request, captured map[string]string, isObjectIDRequired bool) (*PagingRequest, error) {
	var pagingRequest *PagingRequest

	// Paging information...
	switch r.Method {
	case "GET":
		vals := r.URL.Query()
		pr, _ := newPagingRequestFromURLValues(vals)
		pagingRequest = &pr
	case "POST":
		if r.Header.Get("Content-Type") == "application/json" {
			pr, err := newPagingRequestFromJSONBody(r.Body)
			if err != nil {
				return nil, errors.New("Error parsing request from message body")
			}
			pagingRequest = &pr
		} else {
			return nil, errors.New("Unsupported content-type to parse paging information")
		}
	default:
		return nil, errors.New("Unsupported HTTP Method")
	}

	// Look for presence of objectId
	if captured != nil {
		if captured["objectId"] != "" {
			pagingRequest.ObjectID = captured["objectId"]
			_, err := hex.DecodeString(pagingRequest.ObjectID)
			if err != nil {
				return pagingRequest, errors.New("Object Identifier in Request URI is not a hex string")
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

// GetQueryParamAsPositiveInt takes a list of names to check and default value and returns
// the first matching numeric value from the querystring parameters
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

func makeFilterSettingsFromQueryParam(vals url.Values) []FilterSetting {
	rv := []FilterSetting{}

	filterFields := vals["filterField"]
	conditions := vals["condition"]
	expressions := vals["expression"]

	if len(filterFields) > 0 && len(conditions) > 0 && len(expressions) > 0 {
		for i, filterField := range filterFields {
			filterSetting := FilterSetting{}
			filterSetting.FilterField = filterField
			if len(conditions) > i {
				// use condition by ordinal position
				filterSetting.Condition = conditions[i]
			} else {
				// use first/only condition in query
				filterSetting.Condition = conditions[0]
			}
			if len(expressions) > i {
				// use expression by ordinal position
				filterSetting.Expression = expressions[i]
			} else {
				// use first/only expression in query
				filterSetting.Expression = expressions[0]
			}
			rv = append(rv, filterSetting)
		}
	}

	return rv
}

func makeSortSettingsFromQueryParam(vals url.Values) []SortSetting {
	rv := []SortSetting{}

	sortFields := vals["sortField"]
	sortAscendings := vals["sortAscending"]

	if len(sortFields) > 0 {
		for i, sortField := range sortFields {
			sortSetting := SortSetting{}
			sortSetting.SortField = sortField
			if len(sortAscendings) > i {
				sortSetting.SortAscending = (sortAscendings[i] == "true")
			} else {
				if len(sortAscendings) > 0 {
					// use first/only sortAscending in query
					sortSetting.SortAscending = (sortAscendings[0] == "true")
				} else {
					// default
				}
			}
			rv = append(rv, sortSetting)
		}
	}

	return rv
}
