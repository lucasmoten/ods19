package acm

/*
OCAttributeInfo is a nestable structure of ACM modeled after the rawACM on an object
*/
type OCAttributeInfo struct {
	Organizations []string `json:"orgs"`
	Missions      []string `json:"missions"`
	Regions       []string `json:"regions"`
}
