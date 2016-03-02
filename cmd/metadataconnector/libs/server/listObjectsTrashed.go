package server

import (
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/net/context"
)

func (h AppServer) listObjectsTrashed(ctx context.Context, w http.ResponseWriter, r *http.Request) {

	// Get caller value from ctx.
	caller, ok := CallerFromContext(ctx)
	if !ok {
		h.sendErrorResponse(w, 500, errors.New("Could not determine user"), "Invalid user.")
		return
	}
	_ = caller

	// fmt.Fprintf(w, pageTemplateStart, "listObjectsTrashed", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}
