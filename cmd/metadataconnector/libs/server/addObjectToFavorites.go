package server

import (
	"net/http"

	"golang.org/x/net/context"
)

func (h AppServer) addObjectToFavorites(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h.sendErrorResponse(w, 501, nil, "addObjectToFavorites is not yet implemented")
}
