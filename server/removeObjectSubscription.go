package server

import (
	"net/http"

	"golang.org/x/net/context"
)

func (h AppServer) removeObjectSubscription(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return NewAppError(501, nil, "removeObjectSubscription is not yet implemented")
}