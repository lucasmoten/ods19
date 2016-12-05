package server

import (
	"net/http"

	"golang.org/x/net/context"
)

// This endpoint should return a 200 to only denote the availability of a registered odrive.
// This exists because errors at the level of nginx return their own error codes, making it
// ambiguous when trying to determine if at least one odrive is being served up through gatekeeper.
func (h AppServer) ping(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {
	return nil
}
