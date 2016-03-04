package protocol

// ObjectResultset encapsulates the Object defined herein as an array with
// resultset metric information to expose page size, page number, total rows,
// and page count information when retrieving from the data store
type ObjectResultset struct {
	// Resultset contains meta information about the resultset
	Resultset
	// Objects contains the list of objects in this (page of) results.
	Objects []Object `json:"objects,omitempty"`
}
