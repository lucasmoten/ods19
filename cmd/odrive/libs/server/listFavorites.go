package server

import (
	"net/http"

	"golang.org/x/net/context"
)

func (h AppServer) listFavorites(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	sendErrorResponse(&w, 501, nil, "listFavorites is not yet implemented")
}
