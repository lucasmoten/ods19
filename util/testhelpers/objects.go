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
	"testing"

	cfg "decipher.com/object-drive-server/config"

	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/util"
)

// DeferFunc is the function to call with defer
type DeferFunc func()

// GenerateTempFile gives us a file handle for a string that deletes itself on close:
//
//    f, c, err := GenerateTempFile(hugeString)
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

// GenerateTempFileFromBytes creates a file handle from a byte slice, and returns
// a cleanup function. Callers should call `defer` on the function that is returned.
func GenerateTempFileFromBytes(data []byte, t *testing.T) (*os.File, DeferFunc) {
	tmp, err := ioutil.TempFile(".", "__tempfile__")
	if err != nil {
		t.Errorf("GenerateTempFileFromBytes failed. Something is very wrong.")
	}
	tmp.Write(data)
	return tmp, func() {
		name := tmp.Name()
		tmp.Close()
		os.Remove(name)
	}
}

// GenerateEmptyTempFile is for writing
func GenerateEmptyTempFile() (*os.File, DeferFunc, error) {
	tmp, err := ioutil.TempFile(".", "__tempfile__")
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
	permissions[0].AcmShare = fmt.Sprintf(`{"users":[%s]}`, permissions[0].Grantee)
	permissions[0].AcmGrantee.Grantee = permissions[0].Grantee
	permissions[0].AcmGrantee.UserDistinguishedName.String = permissions[0].Grantee
	permissions[0].AcmGrantee.UserDistinguishedName.Valid = true
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
	// TODO: add "share" and set with users or project/groups
	ValidACMUnclassified = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMUnclassifiedEmptyDissemCountries = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":[""],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMUnclassifiedEmptyDissemCountriesEmptyFShare = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":[""],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[""],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	// TODO: Need to figure out what the actual result is and put into f_share
	ValidACMUnclassifiedWithFShare = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":[],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U","banner":"UNCLASSIFIED","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":["cntesttester01oupeopleoudaeouchimeraou_s_governmentcus"],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMUnclassifiedFOUO = `{"version":"2.1.0","classif":"U","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":[],"disponly_to":[""],"dissem_ctrls":["FOUO"],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"U//FOUO","banner":"UNCLASSIFIED//FOUO","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["u"],"f_sci_ctrls":[],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMTopSecretSITK = `{"version":"2.1.0","classif":"TS","owner_prod":[],"atom_energy":[],"sar_id":[],"sci_ctrls":["si","tk"],"disponly_to":[""],"dissem_ctrls":[""],"non_ic":[],"rel_to":[],"fgi_open":[],"fgi_protect":[],"portion":"TS//SI/TK","banner":"TOP SECRET//SI/TK","dissem_countries":["USA"],"accms":[],"macs":[],"oc_attribs":[{"orgs":[],"missions":[],"regions":[]}],"f_clearance":["ts"],"f_sci_ctrls":["si","tk"],"f_accms":[],"f_oc_org":[],"f_regions":[],"f_missions":[],"f_share":[],"f_atom_energy":[],"f_macs":[],"disp_only":""}`

	ValidACMUnclassifiedFOUOSharedToTester01And02 = `{"accms":[],"atom_energy":[],"banner":"UNCLASSIFIED//FOUO","classif":"U","disp_only":"","disponly_to":[""],"dissem_countries":["USA"],"dissem_ctrls":["FOUO"],"f_accms":[],"f_atom_energy":[],"f_clearance":["u"],"f_macs":[],"f_missions":[],"f_oc_org":[],"f_regions":[],"f_sar_id":[],"f_sci_ctrls":[],"f_share":["cntesttester01oupeopleoudaeouchimeraou_s_governmentcus","cntesttester02oupeopleoudaeouchimeraou_s_governmentcus"],"fgi_open":[],"fgi_protect":[],"macs":[],"non_ic":[],"oc_attribs":[{"missions":[],"orgs":[],"regions":[]}],"owner_prod":[],"portion":"U//FOUO","rel_to":[],"sar_id":[],"sci_ctrls":[],"share":{"users":["cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us","cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"]},"version":"2.1.0"}`
)

// Snippets
const (
	SnippetTP10 = "{\"f_macs\":\"{\\\"field\\\":\\\"f_macs\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"tide\\\",\\\"bir\\\",\\\"watchdog\\\"]}\",\"f_oc_org\":\"{\\\"field\\\":\\\"f_oc_org\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"dia\\\"]}\",\"f_accms\":\"{\\\"field\\\":\\\"f_accms\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[]}\",\"f_sap\":\"{\\\"field\\\":\\\"f_sar_id\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_clearance\":\"{\\\"field\\\":\\\"f_clearance\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"ts\\\",\\\"s\\\",\\\"c\\\",\\\"u\\\"]}\",\"f_regions\":\"{\\\"field\\\":\\\"f_regions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_missions\":\"{\\\"field\\\":\\\"f_missions\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[]}\",\"f_share\":\"{\\\"field\\\":\\\"f_share\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"dctc_up2_dctc_manager\\\",\\\"dctc_up2_dctc_supervisor\\\",\\\"dctc_up2_dctc\\\",\\\"dctc_up2_aprc_supervisor\\\",\\\"dctc_up2_aprc_manager\\\",\\\"dctc_up2_aprc\\\",\\\"dctc_up2_administrator\\\",\\\"dctc_watchdog_fle\\\",\\\"dctc_watchdog_sle\\\",\\\"dctc_watchdog_fdo\\\",\\\"dctc_watchdog_user\\\",\\\"dctc_watchdog_administrator\\\",\\\"cntesttester10oupeopleoudaeouchimeraou_s_governmentcus\\\",\\\"cusou_s_governmentouchimeraoudaeoupeoplecntesttester10\\\"]}\",\"f_aea\":\"{\\\"field\\\":\\\"f_atom_energy\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"\\\"]}\",\"f_sci_ctrls\":\"{\\\"field\\\":\\\"f_sci_ctrls\\\",\\\"treatment\\\":\\\"disallow\\\",\\\"values\\\":[\\\"hcs_p\\\",\\\"kdk\\\",\\\"rsv\\\"]}\",\"dissem_countries\":\"{\\\"field\\\":\\\"dissem_countries\\\",\\\"treatment\\\":\\\"allowed\\\",\\\"values\\\":[\\\"USA\\\"]}\"}"
)

// NewCreateObjectPOSTRequest generates a http.Request that will route to the createObject
// controller method, and provide a mutlipart body with the passed-in file object.
// The dn string is optional. The host string is required to route to the correct server,
// e.g. a docker container or localhost. Several object parameters are hardcoded.
// This function should only be used in tests.
func NewCreateObjectPOSTRequest(host, dn string, f *os.File) (*http.Request, error) {
	testName, err := util.NewGUID()
	if err != nil {
		return nil, err
	}

	// TODO change this to object metadata? rjf - that would serialize unwanted zero fields
	var rawAcm interface{}
	json.Unmarshal([]byte(ValidACMUnclassifiedFOUO), &rawAcm)
	createRequest := protocol.CreateObjectRequest{
		Name:     testName,
		TypeName: "File",
		RawAcm:   rawAcm,
	}

	var jsonBody []byte
	jsonBody, err = json.MarshalIndent(createRequest, "", "  ")
	if err != nil {
		return nil, err
	}

	// TODO: we hardcode the name here but the *os.File has associated metadata.
	return NewCreateObjectPOSTRequestRaw("objects", host, dn, f, "testfilename.txt", jsonBody)
}

func NewCreateObjectPOSTRequestRaw(urlPath, host, dn string, f *os.File,
	fileName string, jsonBody []byte) (*http.Request, error) {
	uri := host + cfg.NginxRootURL + "/" + urlPath

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

// NewUpdateObjectStreamPOSTRequest performs an update with the provided protocol.Object as
// metadata. A new stream is posted by generating a small text file with random contents.
func NewUpdateObjectStreamPOSTRequest(t *testing.T, host string, obj protocol.Object) *http.Request {
	uri := host + cfg.NginxRootURL + "/objects/" + obj.ID + "/stream"

	data, _ := util.NewGUID()
	f, closer, err := GenerateTempFile(data)
	if err != nil {
		t.Errorf("could not create tempfile")
		t.FailNow()
	}
	defer closer()

	body, boundary := NewMultipartRequestBody(t, obj, f)

	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		t.Errorf("could not create request")
		t.FailNow()
	}
	req.Header.Set("Content-Type", boundary)

	return req
}

// UpdateObjectStreamPOSTRequest generates a http.Request that will route to the updateObjectStream
// controller method, and provide a mutlipart body with the passed-in file object.
// The dn string is optional. The host string is required to route to the correct server,
// e.g. a docker container or localhost. Several object parameters are hardcoded, and this
// function should only be used for testing purposes.
func UpdateObjectStreamPOSTRequest(id string, changeToken string, host string, dn string, f *os.File) (*http.Request, error) {
	uri := host + cfg.NginxRootURL + "/objects/" + id + "/stream"

	updateRequest := protocol.Object{}
	updateRequest.ID = id
	updateRequest.ChangeToken = changeToken
	updateRequest.RawAcm = ValidACMUnclassifiedFOUO

	jsonBody, err := json.Marshal(updateRequest)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	writePartField(w, "ObjectMetadata", string(jsonBody), "application/json")
	// TODO why do we hardcode the filename here?
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

// NewMultipartRequestBody wraps the creation of a correctly formatted stream of bytes suitable for
// instantiating a http.Request object. The appropriate boundary is also returned, which is required
// to properly set the Content-Type on request headers.
func NewMultipartRequestBody(t *testing.T, obj protocol.Object, f *os.File) (*bytes.Buffer, string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	jsonBody, err := json.Marshal(obj)
	if err != nil {
		t.Errorf("error creating multipart request body: %v", err)
		t.FailNow()
	}

	writePartField(w, "ObjectMetadata", string(jsonBody), "application/json")
	fw, err := w.CreateFormFile("filestream", f.Name())
	if err != nil {
		t.Errorf("error calling CreateFormFile: %v", err)
		t.FailNow()
	}

	// Capture current position of src
	p, err := f.Seek(0, 1)
	if err != nil {
		t.Errorf("error seeking into file %s: %v", f.Name(), err)
		t.FailNow()
	}
	defer func() {
		// Restore position on file when exiting
		f.Seek(p, 0)
	}()
	// Start at beginning for the copy
	f.Seek(0, 0)

	if _, err = io.Copy(fw, f); err != nil {
		t.Errorf("error seeking into file %s into multipart writer: %v", f.Name(), err)
		t.FailNow()
	}
	boundary := w.FormDataContentType()
	w.Close()

	return &b, boundary
}

func NewCreateReadPermissionRequest(obj protocol.Object, grantee, dn, host string) (*http.Request, error) {

	uri := host + cfg.NginxRootURL + "/shared/" + obj.ID
	shareSetting := protocol.ObjectShare{}
	shareString := fmt.Sprintf(`{"users":["%s"]}`, grantee)
	var shareInterface interface{}
	json.Unmarshal([]byte(shareString), &shareInterface)
	shareSetting.Share = shareInterface
	//shareSetting.Grantee = grantee
	shareSetting.AllowRead = true
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
