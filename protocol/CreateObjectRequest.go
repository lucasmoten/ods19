package protocol

// CreateObjectRequest ...
type CreateObjectRequest struct {
	Classification string `json:"classification"`
	Title          string `json:"objectName"`
	FileName       string `json:"fileName"`
	Size           int64  `json:"size"`
	MimeType       string `json:"mimeType"`
}
