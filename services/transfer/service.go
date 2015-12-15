package transfer

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
)

// TransferService ...
type TransferService interface {
	Upload(*multipart.Reader) error
	// Download() error  // TODO
}

// TransferServiceImpl ...
type TransferServiceImpl struct{}

// Upload ...
func (t TransferServiceImpl) Upload(r *multipart.Reader) error {

	// TODO

	return nil

}

// ErrUpload is a standard upload error
var ErrUpload = errors.New("Upload error")

// UploadRequest ...
type UploadRequest struct{}

// UploadResponse ...
type UploadResponse struct {
	Err string `json:"err,omitempty"`
}

// MakeUploadEndpoint ...
func MakeUploadEndpoint(svc TransferService) endpoint.Endpoint {

	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(http.Request)
		reader, err := req.MultipartReader()
		if err != nil {
			return UploadResponse{err.Error()}, nil
		}
		err = svc.Upload(reader)
		if err != nil {
			return UploadResponse{err.Error()}, nil
		}
		return nil, nil

	}

}
