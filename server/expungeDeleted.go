package server

import (
	"encoding/hex"
	"net/http"

	"decipher.com/object-drive-server/events"
	"decipher.com/object-drive-server/mapping"
	"decipher.com/object-drive-server/services/audit"

	"strconv"

	"golang.org/x/net/context"
)

// ExpungedStats just returns the number of objects explicitly expunged
type ExpungedStats struct {
	ExpungedCount int `json:"expunged_count"`
}

// Trash objects that fall into this paging request's page
// We take a paging request to put a bound on the time until a response
func (h AppServer) expungeDeleted(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	// Get user from context
	user, _ := UserFromContext(ctx)
	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "delete"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventDelete")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "DELETE")
	pageSize := 10000
	pageSizeStr := r.URL.Query()["pageSize"]
	if len(pageSizeStr) > 0 {
		var err error
		pageSize, err = strconv.Atoi(pageSizeStr[0])
		if err != nil {
			herr := NewAppError(400, err, "malformed pageSize")
			h.publishError(gem, herr)
			return herr
		}
	}
	expungedObjects, err := dao.ExpungeDeletedByUser(user, pageSize)
	if err != nil {
		herr := NewAppError(500, err, "Unable to expunge deleted objects for user")
		h.publishError(gem, herr)
		return herr
	}
	w.Header().Set("Status", "200")
	for _, o := range expungedObjects.Objects {
		gem = ResetBulkItem(gem)
		gem.Payload.ObjectID = hex.EncodeToString(o.ID)
		gem.Payload.Audit = audit.WithActionTarget(gem.Payload.Audit, NewAuditTargetForID(o.ID))
		gem.Payload.Audit = audit.WithResources(gem.Payload.Audit, NewResourceFromObject(o))
		gem.Payload.ChangeToken = o.ChangeToken
		gem.Payload = events.WithEnrichedPayload(gem.Payload, mapping.MapODObjectToObject(&o))
		h.publishSuccess(gem, w)
	}
	expungedStats := ExpungedStats{ExpungedCount: expungedObjects.TotalRows}
	jsonResponse(w, expungedStats)

	return nil
}
