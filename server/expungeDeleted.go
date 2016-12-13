package server

import (
	"net/http"

	"strconv"

	"golang.org/x/net/context"
)

// ExpungedStats just returns the number of objects explicitly expunged
type ExpungedStats struct {
	ExpungedCount int64 `json:"expunged_count"`
}

// Trash objects that fall into this paging request's page
// We take a paging request to put a bound on the time until a response
func (h AppServer) expungeDeleted(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)
	pageSize := 10000
	pageSizeStr := r.URL.Query()["pageSize"]
	if len(pageSizeStr) > 0 {
		var err error
		pageSize, err = strconv.Atoi(pageSizeStr[0])
		if err != nil {
			return NewAppError(400, err, "malformed pageSize")
		}
	}
	rows, err := dao.ExpungeDeletedByUser(user, pageSize)
	if err != nil {
		return NewAppError(500, err, "Unable to expunge deleted objects for user")
	}
	expungedStats := ExpungedStats{ExpungedCount: rows}
	jsonResponse(w, expungedStats)

	return nil
}
