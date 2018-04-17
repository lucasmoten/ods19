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
		herr := NewAppError(http.StatusInternalServerError, errors.New("could not get capture groups"), "Error parsing URI")
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
	groupobjects := false
	groupname := ""
	var targetObject models.ODObject
	pagingRequest := protocol.PagingRequest{PageSize: 1}
	pagingRequest.FilterSettings = []protocol.FilterSetting{}
	pagingRequest.FilterSettings = append(pagingRequest.FilterSettings, protocol.FilterSetting{FilterField: "name", Condition: "equals", Expression: pathParts[0]})
	for i, part := range pathParts {
		if len(part) > 0 {
			if i == 0 {
				if strings.ToLower(part) != "groupobjects" {
					resultset, err := dao.GetRootObjectsByUser(user, mapping.MapPagingRequestToDAOPagingRequest(&pagingRequest))
					if err != nil {
						herr := NewAppError(http.StatusNotFound, err, "Error finding match at root")
						h.publishError(gem, herr)
						return herr
					}
					if resultset.PageCount > 0 {
						targetObject = resultset.Objects[0]
					} else {
						herr := NewAppError(http.StatusNotFound, err, "No matching objects at root")
						h.publishError(gem, herr)
						return herr
					}
				} else {
					// NOTE: If a user creates a file/folder named 'groupobjects', it wont get served by this as it will be treated as requesting groups
					groupobjects = true
				}
			} else {
				if groupobjects {
					switch i {
					case 1:
						groupname = part
					case 2:
						partFilterSettings := protocol.FilterSetting{FilterField: "name", Condition: "equals", Expression: part}
						pagingRequest.FilterSettings[0] = partFilterSettings
						resultset, err := dao.GetRootObjectsByGroup(groupname, user, mapping.MapPagingRequestToDAOPagingRequest(&pagingRequest))
						if err != nil {
							herr := NewAppError(http.StatusNotFound, err, "Error finding match at groupfolder")
							h.publishError(gem, herr)
							return herr
						}
						if resultset.PageCount > 0 {
							targetObject = resultset.Objects[0]
							// flip back to false as we're no longer at the group root
							groupobjects = false
						} else {
							herr := NewAppError(http.StatusNotFound, err, "No matching objects in groupfolder")
							h.publishError(gem, herr)
							return herr
						}
					}
				} else {
					partFilterSettings := protocol.FilterSetting{FilterField: "name", Condition: "equals", Expression: part}
					pagingRequest.FilterSettings[0] = partFilterSettings
					resultset, err := dao.GetChildObjectsByUser(user, mapping.MapPagingRequestToDAOPagingRequest(&pagingRequest), targetObject)
					if err != nil {
						herr := NewAppError(http.StatusNotFound, err, "Error finding match at folder")
						h.publishError(gem, herr)
						return herr
					}
					if resultset.PageCount > 0 {
						targetObject = resultset.Objects[0]
					} else {
						herr := NewAppError(http.StatusNotFound, err, "No matching objects in folder")
						h.publishError(gem, herr)
						return herr
					}
				}
			}
		} else {
			// path part is empty, effectively ended with /, so this is a folder
			if !groupobjects {
				captured["objectId"] = hex.EncodeToString(targetObject.ID)
				return h.listObjects(context.WithValue(ctx, CaptureGroupsVal, captured), w, r)
			}
			// still focused on groups
			if len(groupname) == 0 {
				// list group info for user
				return h.listMyGroupsWithObjects(ctx, w, r)
			}
			if i == 2 {
				// list objects at root for group
				captured["groupName"] = groupname
				return h.listGroupObjects(context.WithValue(ctx, CaptureGroupsVal, captured), w, r)
			}
			// in a subfolder for group, performs as normal list
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
