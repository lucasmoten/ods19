package server

import (
	"net/http"

	"golang.org/x/net/context"
)

func (h AppServer) getRelationships(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sendErrorResponse(&w, 501, nil, "getRelationships is not yet implemented")
}
