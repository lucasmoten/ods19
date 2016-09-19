package server

import (
	"net/http"

	"golang.org/x/net/context"
)

func (h AppServer) changeOwner(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return NewAppError(501, nil, "changeOwner is not yet implemented")
}
