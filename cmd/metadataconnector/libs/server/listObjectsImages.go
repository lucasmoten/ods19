package server

import (
	"net/http"

	"golang.org/x/net/context"
)

func (h AppServer) listObjectsImages(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sendErrorResponse(&w, 501, nil, "listObjectsImages is not yet implemented")
}
