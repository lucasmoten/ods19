package server

import (
	"net/http"

	"golang.org/x/net/context"
)

func (h AppServer) removeObjectFromFolder(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return NewAppError(501, nil, "removeObjectFromFolder is not yet implemented")
}