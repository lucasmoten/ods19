package protocol

// ObjectLinkResponse is the container for returned data
/*
type ObjectLinkResponse struct {
	TotalRows  int
	PageCount  int
	PageNumber int
	PageSize   int
	PageRows   int
	Objects    []ObjectLink
}

// ObjectLink is the links as exposed to the user of the API
type ObjectLink struct {
	URL         string
	Name        string
	Type        string
	CreateDate  string
	CreatedBy   string
	ChangeToken string
	Size        int64
	ACM         string
}
*/

// ObjectGrant is the grant of an object to a user - possibly the owner
// Granter and URL are implicit in the form of the POST
type ObjectGrant struct {
	Grantee string
	Create  bool
	Read    bool
	Update  bool
	Delete  bool
}
