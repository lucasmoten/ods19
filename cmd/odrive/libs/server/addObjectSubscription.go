package server

import (
	"net/http"

	"golang.org/x/net/context"
)

func (h AppServer) addObjectSubscription(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return NewAppError(501, nil, "addObjectSubscription is not yet implemented")
}
