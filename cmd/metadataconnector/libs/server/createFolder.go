package server

import (
	"decipher.com/oduploader/cmd/metadataconnector/libs/config"
	"decipher.com/oduploader/cmd/metadataconnector/libs/dao"
	"decipher.com/oduploader/metadata/models"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
)

func (h AppServer) createFolder(w http.ResponseWriter, r *http.Request, caller Caller) {
	if r.Method == "POST" {
		var obj models.ODObject
		var acm models.ODACM
		var grant models.ODObjectPermission
		var err error

		// Assume root
		parentID := ""

		r.ParseForm()
		multipartReader, err := r.MultipartReader()
		if err != nil {
			panic(err)
		} // if err != nil
		for {
			part, err := multipartReader.NextPart()
			if err != nil {
				if err == io.EOF {
					break //just an eof...not an error
				} else {
					h.sendErrorResponse(w, 500, err, "error getting a part")
					return
				}
			} // if err != nil

			switch {
			case part.FormName() == "title":
				obj.Name = getFormValueAsString(part)
			case part.FormName() == "type":
				obj.TypeName.String = getFormValueAsString(part)
				obj.TypeName.Valid = (len(obj.TypeName.String) > 0)
			case part.FormName() == "parentId":
				parentID = getFormValueAsString(part)
				if parentID != "" {
					obj.ParentID, err = hex.DecodeString(parentID)
					var p models.ODObject
					p.ID = obj.ParentID
					parentObject, err := dao.GetObject(h.MetadataDB, &p, true)
					if err != nil {
						h.sendErrorResponse(w, 500, err, "error loading parent")
						return
					}
					// Copies permissions from parent. if any
					obj.Permissions = make([]models.ODObjectPermission, len(parentObject.Permissions)+1)
					for pn, objectPermission := range parentObject.Permissions {
						obj.Permissions[pn] = objectPermission
						obj.Permissions[pn].ID = nil
						obj.Permissions[pn].CreatedBy = obj.CreatedBy
					}
				}
			case part.FormName() == "classification":
				acm.Classification.String = getFormValueAsString(part)
				acm.Classification.Valid = (len(acm.Classification.String) > 0)
			} // switch
		} //for

		// Set creator
		obj.CreatedBy = caller.DistinguishedName
		acm.CreatedBy = caller.DistinguishedName
		grant.CreatedBy = caller.DistinguishedName
		grant.Grantee = caller.DistinguishedName
		grant.AllowCreate = true
		grant.AllowRead = true
		grant.AllowUpdate = true
		grant.AllowDelete = true
		if len(obj.Permissions) == 0 {
			obj.Permissions = make([]models.ODObjectPermission, 1)
		}
		obj.Permissions[len(obj.Permissions)-1] = grant

		// TODO: Validation
		// TODO: Access Check

		// Add object to database
		err = dao.CreateObject(h.MetadataDB, &obj, &acm)
		if err != nil {
			h.sendErrorResponse(w, 500, err, "error saving folder")
			return
		}

		// Bounce to redraw the list
		listObjectsURL := config.RootURL
		if len(obj.ParentID) > 0 {
			listObjectsURL += "/object/" + parentID + "/list"
		} else {
			listObjectsURL += "/objects"
		}
		http.Redirect(w, r, listObjectsURL, 301)

	} // if r.Method == "POST" {

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, pageTemplateStart, "createFolder", caller.DistinguishedName)
	fmt.Fprintf(w, pageTemplateEnd)
}
