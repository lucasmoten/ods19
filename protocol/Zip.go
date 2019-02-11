package protocol

// Zip models a request for a list of objects to be zipped into an archive.
type Zip struct {
	// ObjectIds is an array of object identifiers to be compressed in the archive file
	ObjectIDs []string `json:"objectIds"`
	// FileName indicates the file name to assign the generated zip file in the response
	FileName string `json:"fileName"`
	// Disposition indicates whether to return as inline or attachment disposition
	Disposition string `json:"disposition"`
}
