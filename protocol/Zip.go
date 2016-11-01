package protocol

// Zip models a request for a list of objects to be zipped into an archive.
type Zip struct {
	ObjectIDs   []string `json:"objectIds"`
	FileName    string   `json:"fileName"`
	Disposition string   `json:"disposition"`
}
