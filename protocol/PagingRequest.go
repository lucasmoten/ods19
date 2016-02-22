package protocol

// PagingRequest supports a request constrained to a given page number and size
type PagingRequest struct {
	// PageNumber is the requested page number for this request
	PageNumber int
	// PageSize is the requested page size for this request
	PageSize int
	// ParentID if provided lets us list the children
	ParentID string `json:"parentId"`
}
