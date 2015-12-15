package transfer

import (
	"errors"
	"io"
	"log"
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

	log.Println("got this reader", r)
	var partBytes int
	for {
		part, partErr := r.NextPart()
		if partErr != nil {
			if partErr == io.EOF {
				break //just an eof...not an error
			} else {
				log.Printf("error getting a part %v", partErr)
				// http.Error(w, "error getting a part", 500)
				return partErr
			}
		} else {
			if len(part.FileName()) > 0 {
				//Could take an *indefinite* amount of time!!
				buffer := make([]byte, 1024)
				bytesRead, err := part.Read(buffer)
				if err != nil {
					return err
				}
				partBytes += bytesRead
			}

		}
	}
	log.Printf("Read this many bytes without error", partBytes)
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

		req := request.(*http.Request)
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
