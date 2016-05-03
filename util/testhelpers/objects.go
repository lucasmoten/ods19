package testhelpers

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"

	cfg "decipher.com/object-drive-server/config"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

// DeferFunc is the function to call with defer
type DeferFunc func()

// GenerateTempFile gives us a file handle for a string that deletes itself on close:
//
//    f,c,err := GenerateTempFile(hugeString)
//    if err != nil {
//      return err
//    }
//    defer c()
//
func GenerateTempFile(data string) (*os.File, DeferFunc, error) {
	tmp, err := ioutil.TempFile(".", "__tempfile__")
	tmp.WriteString(data)
	return tmp, func() {
		name := tmp.Name()
		tmp.Close()
		err = os.Remove(name)
	}, err
}

// DoWithDecodedResult is the common case of getting back a json response that is ok
func DoWithDecodedResult(client *http.Client, req *http.Request) (*http.Response, interface{}, error) {
	var objResponse protocol.Object
	res, err := client.Do(req)
	if err != nil {
		return nil, objResponse, err
	}
	err = util.FullDecode(res.Body, &objResponse)
	res.Body.Close()
	return res, objResponse, err
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
	obj.RawAcm.String = ValidACMUnclassified
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

	permissions := make([]models.ODObjectPermission, 1)
	permissions[0].Grantee = username
	permissions[0].AllowCreate = true
	permissions[0].AllowRead = true
	permissions[0].AllowUpdate = true
	permissions[0].AllowDelete = true
	obj.Permissions = permissions

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

// ValidACMs
const (
	ValidACMUnclassified = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMUnclassifiedFOUO = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":["FOUO"],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U//FOUO","banner":"UNCLASSIFIED//FOUO","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMTopSecretSITK = `{"version":"2.1.0","classif":"TS","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":["si","tk"],"disponly_to":[""],"dissem_ctrls":[""],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"TS//SI/TK","banner":"TOP SECRET//SI/TK","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["ts"],"f_sci_ctrls":["si","tk"],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`
)

// NewCreateObjectPOSTRequest generates a http.Request that will route to the createObject
// controller method, and provide a mutlipart body with the passed-in file object.
// The dn string is optional. The host string is required to route to the correct server,
// e.g. a docker container or localhost. Several object parameters are hardcoded, and this
// function should only be used for testing purposes.
func NewCreateObjectPOSTRequest(host, dn string, f *os.File) (*http.Request, error) {
	testName, err := util.NewGUID()
	if err != nil {
		return nil, err
	}

	// TODO change this to object metadata? rjf - that would serialize unwanted zero fields
	createRequest := protocol.CreateObjectRequest{
		Name:     testName,
		TypeName: "File",
		RawAcm:   ValidACMUnclassifiedFOUO,
	}

	var jsonBody []byte
	jsonBody, err = json.Marshal(createRequest)
	if err != nil {
		return nil, err
	}

	req, err := NewCreateObjectPOSTRequestRaw(
		"objects",
		host, dn,
		f,
		"testfilename.txt",
		jsonBody,
	)
	return req, err
}

// NewCreateObjectPOSTRequestRaw generates a raw request, with enough flexibility to make
// some malformed requests without too much trouble.
func NewCreateObjectPOSTRequestRaw(
	requestType,
	host, dn string,
	f *os.File,
	fileName string,
	jsonBody []byte,
) (*http.Request, error) {
	uri := host + cfg.NginxRootURL + "/" + requestType

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	writePartField(w, "ObjectMetadata", string(jsonBody), "application/json")
	fw, err := w.CreateFormFile("filestream", fileName)
	if err != nil {
		return nil, err
	}

	// Capture current position of src
	p, err := f.Seek(0, 1)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Restore position on file when exiting
		f.Seek(p, 0)
	}()
	// Start at beginning for the copy
	f.Seek(0, 0)

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

// UpdateObjectStreamPOSTRequest generates a http.Request that will route to the updateObjectStream
// controller method, and provide a mutlipart body with the passed-in file object.
// The dn string is optional. The host string is required to route to the correct server,
// e.g. a docker container or localhost. Several object parameters are hardcoded, and this
// function should only be used for testing purposes.
func UpdateObjectStreamPOSTRequest(id string, changeToken string, host string, dn string, f *os.File) (*http.Request, error) {
	uri := host + cfg.NginxRootURL + "/objects/" + id + "/stream"

	updateRequest := protocol.UpdateStreamRequest{
		ChangeToken: changeToken,
		RawAcm:      ValidACMUnclassifiedFOUO,
	}
	jsonBody, err := json.Marshal(updateRequest)
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

	// Capture current position of src
	p, err := f.Seek(0, 1)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Restore position on file when exiting
		f.Seek(p, 0)
	}()
	// Start at beginning for the copy
	f.Seek(0, 0)

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

func NewCreateReadPermissionRequest(obj protocol.Object, grantee, dn, host string) (*http.Request, error) {

	uri := host + cfg.NginxRootURL + "/shared/" + obj.ID
	shareSetting := protocol.ObjectGrant{}
	shareSetting.Grantee = grantee
	shareSetting.Read = true
	jsonBody, err := json.Marshal(shareSetting)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if dn != "" {
		req.Header.Set("USER_DN", dn)
	}
	return req, nil
}

func NewDeletePermissionRequest(obj protocol.Object, share protocol.Permission, dn, host string) (*http.Request, error) {
	uri := host + cfg.NginxRootURL + "/shared/" + obj.ID + "/" + share.ID
	removeSetting := protocol.RemoveObjectShareRequest{}
	removeSetting.ObjectID = obj.ID
	removeSetting.ShareID = share.ID
	removeSetting.ChangeToken = share.ChangeToken
	jsonBody, err := json.Marshal(removeSetting)
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

// NewDeleteObjectRequest creates an http.Request that will route to the deleteObject
// controller method. The dn parameter is optional and in most cases does not need to be
// set. Host must be provided to route to the correct server, e.g. a docker container or
// localhost.
func NewDeleteObjectRequest(obj protocol.Object, dn, host string) (*http.Request, error) {

	uri := host + cfg.NginxRootURL + "/objects/" + obj.ID + "/trash"

	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = obj.ChangeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
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

	uri := host + cfg.NginxRootURL + "/objects/" + id + "/properties"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// New GetObjectStreamRequest ...
func NewGetObjectStreamRequest(id, dn, host string) (*http.Request, error) {

	uri := host + cfg.NginxRootURL + "/objects/" + id + "/stream"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// New GetObjectStreamRevisionRequest ...
func NewGetObjectStreamRevisionRequest(id string, version string, dn string, host string) (*http.Request, error) {

	uri := host + cfg.NginxRootURL + "/revisions/" + id + "/" + version + "/stream"

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewUndeleteObjectPUTRequest creates a request with the provided objectID in the URI
// that routes to the removeObjectFromTrash handler.
func NewUndeleteObjectDELETERequest(id, changeToken, dn, host string) (*http.Request, error) {
	if id == "" {
		return nil, errors.New("Test ObjectID cannot be empty string")
	}

	uri := host + cfg.NginxRootURL + "/objects/" + id + "/untrash"

	objChangeToken := protocol.ChangeTokenStruct{}
	objChangeToken.ChangeToken = changeToken
	jsonBody, err := json.Marshal(objChangeToken)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
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

// AreFilesTheSame checks if the contents of the file hash to the same MD5
func AreFilesTheSame(file1 *os.File, file2 *os.File) bool {

	// Get hashes
	h1, err := hashMD5OfFile(file1)
	if err != nil {
		log.Printf("Error getting hash of file 1: %v", err)
		return false
	}
	h2, err := hashMD5OfFile(file2)
	if err != nil {
		log.Printf("Error getting hash of file 2: %v", err)
		return false
	}

	return (bytes.Compare(h1, h2) == 0)
}

const filechunk = 8192

func hashMD5OfFile(file *os.File) ([]byte, error) {

	// Capture current position
	p, err := file.Seek(0, 1)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Restore position on file when exiting
		file.Seek(p, 0)
	}()
	// Start at beginning for processing
	file.Seek(0, 0)

	// calculate the file size
	info, _ := file.Stat()
	filesize := info.Size()
	blocks := uint64(math.Ceil(float64(filesize) / float64(filechunk)))
	hash := md5.New()

	for i := uint64(0); i < blocks; i++ {
		blocksize := int(math.Min(filechunk, float64(filesize-int64(i*filechunk))))
		buf := make([]byte, blocksize)

		file.Read(buf)
		io.WriteString(hash, string(buf)) // append into the hash
	}

	return hash.Sum(nil), nil
}
