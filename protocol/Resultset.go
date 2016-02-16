package protocol

// Resultset provides a summation of an accompanying array of items for which
// it refers from a request with a page number and size.  For example, if the
// request is for page 3 of widgets, with 20 returned per page, and there are 56
// widgets that match, then TotalRows=56, PageCount=3, PageNumber=3, PageSize=20
// and PageRows=16
type Resultset struct {
	// TotalRows is the total number of items matching the same query resulting
	// in this page of results
	TotalRows int
	// PageCount is the total rows divided by page size
	PageCount int
	// PageNumber is the requested page number for this resultset
	PageNumber int
	// PageSize is the requested page size for this resultset
	PageSize int
	// PageRows is the number of items included in this page of the results, which
	// may be less than pagesize, but never greater.
	PageRows int
}
