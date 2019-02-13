package protocol

// Breadcrumb defines a mimimal set of data for clients to display or link to an object's
// parent chain. To get all of a breadcrumb's properties, get the properties associated
// with a breadcrumb's ID.
type Breadcrumb struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// ParentID references another Object by its ID indicating which object, if
	// any, contains, or is an ancestor of this object. (e.g., folder). An object
	// without a parent is considered to be contained within the 'root' or at the
	// 'top level'.
	ParentID string `json:"parentId"`
	// Name is the given name for the object. (e.g., filename)
	Name string `json:"name"`
}
