package testhelpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"

	"decipher.com/oduploader/metadata/models"
	"decipher.com/oduploader/protocol"
	"decipher.com/oduploader/util"
)

func NewACMForUser(username, classification string) models.ODACM {
	var acm models.ODACM
	acm.CreatedBy = username
	acm.Classification.String = classification
	acm.Classification.Valid = true
	return acm
}

// NewObjectWithPermissionsAndProperties creates a single minimally populated
// object with random properties and full permissions.
func NewObjectWithPermissionsAndProperties(username, objectType string) models.ODObject {

	var obj models.ODObject
	randomName, err := util.NewGUID()
	if err != nil {
		panic(err)
	}

	obj.Name = randomName
	obj.CreatedBy = username
	obj.TypeName.String, obj.TypeName.Valid = objectType, true
	permissions := make([]models.ODObjectPermission, 1)
	permissions[0].Grantee = obj.CreatedBy
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	obj.Permissions = permissions
	properties := make([]models.ODObjectPropertyEx, 1)
	properties[0].Name = "Test Property for " + randomName
	properties[0].Value.String = "Property Val for " + randomName
	properties[0].Value.Valid = true
	properties[0].ClassificationPM.String = "UNCLASSIFIED"
	properties[0].ClassificationPM.Valid = true
	obj.Properties = properties

	return obj
}

// NewTrashedObject creates a deleted object owned by the passed in user.
// There are no database calls in this function.
func NewTrashedObject(username string) models.ODObject {
	var obj models.ODObject
	obj.IsDeleted = true
	obj.OwnedBy.String, obj.OwnedBy.Valid = username, true

	name, _ := util.NewGUID()
	obj.Name = name

	return obj
}

// CreateParentChildObjectRelationship sets the ParentID of child to the ID of parent.
// If parent has no ID, a []byte GUID is generated.
func CreateParentChildObjectRelationship(parent, child models.ODObject) (models.ODObject, models.ODObject, error) {

	if len(parent.ID) == 0 {
		id, err := util.NewGUIDBytes()
		if err != nil {
			return parent, child, err
		}
		parent.ID = id
	}
	child.ParentID = parent.ID
	return parent, child, nil
}

// NewCreateObjectPOSTRequest generates a http.Request that will route to the createObject
// controller method, and provide a mutlipart body with the passed-in file object.
// The dn string is optional. The host string is required to route to the correct server,
// e.g. a docker container or localhost. Several object parameters are hardcoded, and this
// function should only be used for testing purposes.
func NewCreateObjectPOSTRequest(host, dn string, f *os.File) (*http.Request, error) {
	uri := host + TestServicePrefix + "object"
	testName, err := util.NewGUID()
	if err != nil {
		return nil, err
	}

	// TODO change this to object metadata?
	createRequest := protocol.CreateObjectRequest{
		Name:     testName,
		TypeName: "File",
		RawAcm:   `{"version":"2.1.0","classif":"S"}`,
	}
	jsonBody, err := json.Marshal(createRequest)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	writePartField(w, "ObjectMetadata", string(jsonBody), "application/json")
	fw, err := w.CreateFormFile("filestream", "testfilename.txt")
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return nil, err
	}
	w.Close()

	req, err := http.NewRequest("POST", uri, &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Type", w.FormDataContentType())
	if dn != "" {
		req.Header.Set("USER_DN", dn)
	}

	return req, nil
}

// NewDeleteObjectRequest creates an http.Request that will route to the deleteObject
// controller method. The dn parameter is optional and in most cases does not need to be
// set. Host must be provided to route to the correct server, e.g. a docker container or
// localhost.
func NewDeleteObjectRequest(obj protocol.Object, dn, host string) (*http.Request, error) {

	uri := host + TestServicePrefix + "object/" + obj.ID

	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = obj.ChangeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("DELETE", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if dn != "" {
		req.Header.Set("USER_DN", dn)
	}
	return req, nil
}

// NewGetObjectRequest ...
func NewGetObjectRequest(id, dn, host string) (*http.Request, error) {

	uri := host + TestServicePrefix + "object/" + id + "/properties"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewUndeleteObjectPUTRequest creates a request with the provided objectID in the URI
// that routes to the removeObjectFromTrash handler.
func NewUndeleteObjectPUTRequest(id, changeToken, dn, host string) (*http.Request, error) {
	if id == "" {
		return nil, errors.New("Test ObjectID cannot be empty string")
	}

	uri := host + TestServicePrefix + "trash/" + id

	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = changeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	if dn != "" {
		req.Header.Set("USER_DN", dn)
	}
	return req, nil
}

func writePartField(w *multipart.Writer, fieldname, value, contentType string) error {
	p, err := createFormField(w, fieldname, contentType)
	if err != nil {
		return err
	}
	_, err = p.Write([]byte(value))
	return err
}

func createFormField(w *multipart.Writer, fieldname, contentType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"`, escapeQuotes(fieldname)))
	h.Set("Content-Type", contentType)
	return w.CreatePart(h)
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
