package protocol

// Breadcrumb defines a mimimal set of data for clients to display or link to an object's
// parent chain. To get all of a breadcrumb's properties, get the properties associated
// with a breadcrumb's ID.
type Breadcrumb struct {
	ID       string `json:"id"`
	ParentID string `json:"parentId"`
	Name     string `json:"name"`
}
