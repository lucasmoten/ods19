package server

import (
	"net/http"

	"golang.org/x/net/context"
)

func (h AppServer) removeObjectFromFolder(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h.sendErrorResponse(w, 501, nil, "removeObjectFromFolder is not yet implemented")
}
