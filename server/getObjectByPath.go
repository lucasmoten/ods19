package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/context"

	"github.com/deciphernow/object-drive-server/mapping"
	"github.com/deciphernow/object-drive-server/metadata/models"
	"github.com/deciphernow/object-drive-server/protocol"
	"github.com/deciphernow/object-drive-server/services/audit"
)

func (h AppServer) getObjectByPath(ctx context.Context, w http.ResponseWriter, r *http.Request) *AppError {

	dao := DAOFromContext(ctx)
	gem, _ := GEMFromContext(ctx)
	gem.Action = "access"
	gem.Payload.Audit = audit.WithType(gem.Payload.Audit, "EventAccess")
	gem.Payload.Audit = audit.WithAction(gem.Payload.Audit, "ACCESS")

	// Get capture groups from ctx.
	captured, ok := CaptureGroupsFromContext(ctx)
	if !ok {
		herr := NewAppError(500, errors.New("could not get capture groups"), "Error parsing URI")
		h.publishError(gem, herr)
		return herr
	}

	path := captured["path"]
	if path == "" {
		// Request for listing root objects
		return h.listObjects(ctx, w, r)
	}

	path, _ = url.PathUnescape(path)
	pathParts := strings.Split(path, "/")
	user, _ := UserFromContext(ctx)
	var targetObject models.ODObject
	pagingRequest := protocol.PagingRequest{PageSize: 1}
	pagingRequest.FilterSettings = []protocol.FilterSetting{}
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, protocol.FilterSetting{FilterField: "name", Condition: "equals", Expression: pathParts[0]})
	for i, part := range pathParts {
		if len(part) > 0 {
			if i == 0 {
				resultset, err := dao.GetRootObjectsByUser(user, mapping.MapPagingRequestToDAOPagingRequest(&pagingRequest))
				if err != nil {
					herr := NewAppError(404, err, "Error finding match at root")
					h.publishError(gem, herr)
					return herr
				}
				if resultset.PageCount > 0 {
					targetObject = resultset.Objects[0]
				} else {
					herr := NewAppError(404, err, "No matching objects at root")
					h.publishError(gem, herr)
					return herr
				}
			} else {
				partFilterSettings := protocol.FilterSetting{FilterField: "name", Condition: "equals", Expression: pathParts[i]}
				pagingRequest.FilterSettings[0] = partFilterSettings
				resultset, err := dao.GetChildObjectsByUser(user, mapping.MapPagingRequestToDAOPagingRequest(&pagingRequest), targetObject)
				if err != nil {
					herr := NewAppError(404, err, "Error finding match at folder")
					h.publishError(gem, herr)
					return herr
				}
				if resultset.PageCount > 0 {
					targetObject = resultset.Objects[0]
				} else {
					herr := NewAppError(404, err, "No matching objects in folder")
					h.publishError(gem, herr)
					return herr
				}
			}
		} else {
			// path ended with /, so this is a folder
			//captured := make(map[string]string)
			captured["objectId"] = hex.EncodeToString(targetObject.ID)
			return h.listObjects(context.WithValue(ctx, CaptureGroupsVal, captured), w, r)
		}
	}

	// path did not end with /, so this may be a file
	if targetObject.ContentSize.Int64 > 0 {
		captured["objectId"] = hex.EncodeToString(targetObject.ID)
		return h.getObjectStream(context.WithValue(ctx, CaptureGroupsVal, captured), w, r)
	}
	// treat as a folder, listing children
	captured["objectId"] = hex.EncodeToString(targetObject.ID)
	return h.listObjects(context.WithValue(ctx, CaptureGroupsVal, captured), w, r)

}
