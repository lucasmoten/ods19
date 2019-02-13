package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/deciphernow/gm-fabric-go/tlsutil"
)

// ObjectDrive defines operations for our client.
type ObjectDrive interface {
	ChangeOwner(ChangeOwnerRequest) (Object, error)
	CopyObject(CopyObjectRequest) (Object, error)
	CreateObject(CreateObjectRequest, io.Reader) (Object, error)
	DeleteObject(id string, token string) (DeletedObjectResponse, error)
	ExpungeObject(id string, token string) (ExpungedObjectResponse, error)
	GetHttpClient() *http.Client
	GetObject(id string) (Object, error)
	GetObjectStream(id string) (io.ReadCloser, error)
	GetRevisions(id string) (ObjectResultset, error)
	MoveObject(MoveObjectRequest) (Object, error)
	RestoreRevision(id string, token string, changeCount int) (Object, error)
	Search(paging PagingRequest, searchAllObjects bool) (ObjectResultset, error)
	UpdateObject(UpdateObjectRequest) (Object, error)
	UpdateObjectAndStream(UpdateObjectAndStreamRequest, io.Reader) (Object, error)
}

// Client implements ObjectDrive.
type Client struct {
	httpClient *http.Client
	url        string
	// Verbose will print extra debug information if true.
	Verbose bool
	Conf    Config
	MyDN    string
}

// Verify that Client Implements ObjectDrive.
var _ ObjectDrive = (*Client)(nil)

// Config defines the bare minimum that must be statically configured for a Client.
type Config struct {
	Cert       string
	Trust      string
	Key        string
	SkipVerify bool // DO NOT SET THIS.  Set ServerName to match CN of the Remote
	// Remote specifies the full API proxy prefix: https://{host}:{port}/{prefix}
	// Actual object drive API endpoints are appended to this string.
	Remote string
	// Impersonation is a DN of a user we want to impersonate. If set, HTTP headers
	// USER_DN will be set to this value, and EXTERNAL_SYS_DN and SSL_CLIENT_S_DN
	// will be set to the Client.MyDN field.
	Impersonation string
	ServerName    string // OD_PEER_CN has this value, or if it's blank then it must match Dial hostname
}

// NewClient instantiates a new Client that implements ObjectDrive.  This client can be used to perform
// CRUD operations on a running ObjectDrive instance.
//
// The client requires a configuration structure that contains the key bits of information necessary to
// establish a connection to the ObjectDrive: certificates, trusts, keys, and remote URL.
func NewClient(conf Config) (*Client, error) {
	trust, err := ioutil.ReadFile(conf.Trust)
	if err != nil {
		return nil, fmt.Errorf("while opening trust file %s: %v", conf.Trust, err)
	}
	caPool := x509.NewCertPool()
	if caPool.AppendCertsFromPEM(trust) == false {
		if len(trust) > 0 {
			return nil, fmt.Errorf("while appending certs in trust file %s", conf.Trust)
		}
		return nil, fmt.Errorf("no certificates listed in trust file %s", conf.Trust)
	}
	cert, err := tls.LoadX509KeyPair(conf.Cert, conf.Key)
	if err != nil {
		return nil, fmt.Errorf("while opening cert and key file %s, %s: %v", conf.Cert, conf.Key, err)
	}

	pub, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("while parsing public certificate from cert and key file %s, %s: %v", conf.Cert, conf.Key, err)
	}
	mydn := tlsutil.GetDistinguishedName(pub)

	tlsConfig := &tls.Config{
		InsecureSkipVerify:       conf.SkipVerify,
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                caPool,
		RootCAs:                  caPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS10,
		ServerName:               conf.ServerName,
	}
	tlsConfig.BuildNameToCertificate()

	var c http.Client
	c.Transport = &http.Transport{TLSClientConfig: tlsConfig}

	return &Client{&c, conf.Remote, false, conf, mydn}, nil
}

// ChangeOwner changes the object's ownedBy field.
func (c *Client) ChangeOwner(req ChangeOwnerRequest) (Object, error) {
	uri := c.url + "/objects/" + req.ID + "/owner/" + req.NewOwner
	var ret Object

	resp, err := c.doPost(uri, req)
	if err != nil {
		return ret, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, fmt.Errorf("could not decode response: %v", err)
	}

	return ret, nil
}

// CopyObject performs the copy operation on the ObjectDrive from the CiotObjectRequest that indicates
// the object to be copied by referencing its id. The caller must have permission to the object and its
// revisions. The resultant object is created as a sibling to the original. File streams, if any,
// and accompanied permissions, derive from the original object
func (c *Client) CopyObject(req CopyObjectRequest) (Object, error) {
	uri := c.url + "/objects/" + req.ID + "/copy"
	var ret Object

	resp, err := c.doPost(uri, nil)
	if err != nil {
		return ret, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	if c.Verbose {
		data, _ := httputil.DumpResponse(resp, true)
		fmt.Printf("%s", string(data))
	}

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	// Send back the created object properties
	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

// CreateObject performs the create operation on the ObjectDrive from the CreateObjectRequest that fully
// specifies the object to be created.  The caller must also provide an open io.Reader interface to the stream
// they wish to upload.  If creating an object with no filestream, such as a folder, then reader must be nil.
func (c *Client) CreateObject(req CreateObjectRequest, reader io.Reader) (Object, error) {
	uri := c.url + "/objects"
	var ret Object

	jsonBody, err := json.MarshalIndent(req, "", "    ")
	if err != nil {
		return ret, fmt.Errorf("could not marshal json: %v", err)
	}

	var body bytes.Buffer
	var contentType string

	// If an io.Reader is passed, upload its contents.
	if reader != nil {
		writer := multipart.NewWriter(&body)

		writePartField(writer, "ObjectMetadata", string(jsonBody), "application/json")
		part, err := writer.CreateFormFile("filestream", strings.TrimSpace(req.Name))
		if err != nil {
			return ret, err
		}

		if _, err = io.Copy(part, reader); err != nil {
			return ret, err
		}

		err = writer.Close()
		if err != nil {
			return ret, err
		}

		contentType = writer.FormDataContentType()
	} else {
		body.Write([]byte(jsonBody))
	}

	httpReq, err := http.NewRequest("POST", uri, &body)
	if err != nil {
		return ret, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	if c.Conf.Impersonation != "" {
		setImpersonationHeaders(httpReq, c.Conf.Impersonation, c.MyDN)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.Println(err)
		return ret, err
	}
	defer resp.Body.Close()
	if c.Verbose {
		data, _ := httputil.DumpResponse(resp, true)
		fmt.Printf("%s", string(data))
	}

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	// Send back the created object properties
	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

// DeleteObject moves an object on the server to the trash.  The object's ID and changetoken from the
// current object in ObjectDrive are needed to perform the operation.
func (c *Client) DeleteObject(id string, token string) (DeletedObjectResponse, error) {

	url := c.url + "/objects/" + id + "/trash"

	var ret DeletedObjectResponse

	deleteRequest := DeleteObjectRequest{
		ID:          id,
		ChangeToken: token,
	}

	resp, err := c.doPost(url, deleteRequest)
	if err != nil {
		log.Println(err)
		return ret, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

// ExpungeObject deletes an object from the system. It cannot be restored from the user's trash
// using the API calls
func (c *Client) ExpungeObject(id string, token string) (ExpungedObjectResponse, error) {

	url := c.url + "/objects/" + id

	var ret ExpungedObjectResponse

	deleteRequest := DeleteObjectRequest{
		ID:          id,
		ChangeToken: token,
	}

	resp, err := c.doDelete(url, deleteRequest)
	if err != nil {
		log.Println(err)
		return ret, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

func (c *Client) GetHttpClient() *http.Client {
	return c.httpClient
}

// GetObject fetches the metadata associated with an object by its unique ID.
func (c *Client) GetObject(id string) (Object, error) {
	var ret Object

	propertyURL := c.url + "/objects/" + id + "/properties"

	resp, err := c.doGet(propertyURL, nil)
	if err != nil {
		return ret, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ret, err
	}

	jsonErr := json.Unmarshal(body, &ret)
	if jsonErr != nil {
		return ret, jsonErr
	}

	return ret, nil
}

// GetObjectStream fetches the filestream associated with an object, if any exists.
func (c *Client) GetObjectStream(id string) (io.ReadCloser, error) {
	fileURL := c.url + "/objects/" + id + "/stream"

	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		return nil, err
	}

	if c.Conf.Impersonation != "" {
		setImpersonationHeaders(req, c.Conf.Impersonation, c.MyDN)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("http status: %v", resp.StatusCode)
	}

	return resp.Body, nil
}

// GetRevisions fetches the revisions over time for an object by its unique ID.
func (c *Client) GetRevisions(id string) (ObjectResultset, error) {
	var ret ObjectResultset

	revisionURL := c.url + "/revisions/" + id

	resp, err := c.doGet(revisionURL, nil)
	if err != nil {
		return ret, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ret, err
	}

	jsonErr := json.Unmarshal(body, &ret)
	if jsonErr != nil {
		return ret, jsonErr
	}

	return ret, nil
}

// MoveObject moves a given file or folder into a new parent folder, both specified by ID.
func (c *Client) MoveObject(req MoveObjectRequest) (Object, error) {
	uri := c.url + "/objects/" + req.ID + "/move/" + req.ParentID
	var ret Object

	resp, err := c.doPost(uri, req)
	if err != nil {
		return ret, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, fmt.Errorf("could not decode response: %v", err)
	}

	return ret, nil
}

// Ping checks if the server is up
func (c *Client) Ping() (bool, error) {
	pingURL := c.url + "/ping"
	req, err := http.NewRequest("GET", pingURL, nil)
	if err != nil {
		return false, fmt.Errorf("error creating request: %v", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()
	return (resp.StatusCode == http.StatusOK), nil
}

// RestoreRevision is a convenience operation to restore a prior version as the current version
// without the need to reupload a file, pass permissions or properties.  The permissions and ACM
// will remain the same as the current version, and this operation is not permitted if the owner
// or location would change.
func (c *Client) RestoreRevision(id string, token string, changeCount int) (Object, error) {

	uri := c.url + "/revisions/" + id + "/" + strconv.Itoa(changeCount) + "/restore"
	var ret Object

	req := ChangeTokenStruct{
		ChangeToken: token,
	}

	resp, err := c.doPost(uri, req)
	if err != nil {
		return ret, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, fmt.Errorf("could not decode response: %v", err)
	}

	return ret, nil
}

// Search facilitates listing objects at root, under a folder, or full breadth
// search of all objects applying the requesting filtering, sorting, and paging
// conditions to the results
func (c *Client) Search(paging PagingRequest, searchAllObjects bool) (ObjectResultset, error) {
	uri := c.url
	if searchAllObjects {
		uri += "/search/using-client-library"
	} else {
		uri += "/objects"
		if len(paging.ObjectID) > 0 {
			uri += "/" + url.QueryEscape(paging.ObjectID)
		}
	}
	uri += "?"
	if len(paging.FilterMatchType) > 0 {
		uri += fmt.Sprintf("filterMatchType=%s&", url.QueryEscape(paging.FilterMatchType))
	}
	for _, fs := range paging.FilterSettings {
		uri += fmt.Sprintf("filterField=%s&condition=%s&expression=%s&", url.QueryEscape(fs.FilterField), url.QueryEscape(fs.Condition), url.QueryEscape(fs.Expression))
	}
	for _, ss := range paging.SortSettings {
		uri += fmt.Sprintf("sortField=%s&sortAscending=%t&", url.QueryEscape(ss.SortField), ss.SortAscending)
	}
	uri += fmt.Sprintf("pageNumber=%d&pageSize=%d&", paging.PageNumber, paging.PageSize)

	var ret ObjectResultset
	resp, err := c.doGet(uri, nil)
	if err != nil {
		return ret, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ret, err
	}

	jsonErr := json.Unmarshal(body, &ret)
	if jsonErr != nil {
		return ret, jsonErr
	}

	return ret, nil
}

// UpdateObject updates an object's metadata or permissions. To update an actual
// filestream, use UpdateObjectAndStream.
func (c *Client) UpdateObject(req UpdateObjectRequest) (Object, error) {
	uri := c.url + "/objects/" + req.ID + "/properties"
	var ret Object

	resp, err := c.doPost(uri, req)
	if err != nil {
		return ret, fmt.Errorf("http error %v: %v", resp.StatusCode, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, fmt.Errorf("could not decode response: %v", err)
	}

	return ret, nil
}

// UpdateObjectAndStream updates an object's associated stream as well as its metadata or permissions.
func (c *Client) UpdateObjectAndStream(req UpdateObjectAndStreamRequest, r io.Reader) (Object, error) {
	uri := c.url + "/objects/" + req.ID + "/stream"
	var ret Object

	if r == nil {
		return ret, errors.New("you must provide an io.Reader")
	}

	jsonBody, err := json.MarshalIndent(req, "", "    ")
	if err != nil {
		return ret, fmt.Errorf("could not marshal json: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writePartField(writer, "ObjectMetadata", string(jsonBody), "application/json")
	part, err := writer.CreateFormFile("filestream", req.Name)
	if err != nil {
		return ret, err
	}
	contentType := writer.FormDataContentType()

	if _, err = io.Copy(part, r); err != nil {
		return ret, err
	}

	err = writer.Close()
	if err != nil {
		return ret, err
	}

	httpReq, err := http.NewRequest("POST", uri, &body)
	if err != nil {
		return ret, err
	}

	httpReq.Header.Set("Content-Type", contentType)

	if c.Conf.Impersonation != "" {
		setImpersonationHeaders(httpReq, c.Conf.Impersonation, c.MyDN)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.Println(err)
		return ret, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ret, errorFromResponse(resp)
	}

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, fmt.Errorf("could not decode response: %v", err)
	}

	return ret, nil
}

func (c *Client) doDelete(uri string, body interface{}) (*http.Response, error) {
	return c.doMethod("DELETE", uri, body)
}
func (c *Client) doGet(uri string, body interface{}) (*http.Response, error) {
	return c.doMethod("GET", uri, body)
}
func (c *Client) doPatch(uri string, body interface{}) (*http.Response, error) {
	return c.doMethod("POST", uri, body)
}
func (c *Client) doPost(uri string, body interface{}) (*http.Response, error) {
	return c.doMethod("POST", uri, body)
}
func (c *Client) doPut(uri string, body interface{}) (*http.Response, error) {
	return c.doMethod("POST", uri, body)
}
func (c *Client) doMethod(method string, uri string, body interface{}) (*http.Response, error) {
	var err error
	var jsonBody []byte
	var req *http.Request
	if body != nil {
		jsonBody, err = json.MarshalIndent(body, "", "    ")
		if err != nil {
			return nil, fmt.Errorf("could not marshall json body: %v", err)
		}
		req, err = http.NewRequest(method, uri, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, uri, nil)
	}
	if err != nil {
		return nil, err
	}

	if c.Conf.Impersonation != "" {
		setImpersonationHeaders(req, c.Conf.Impersonation, c.MyDN)
	}

	return c.httpClient.Do(req)
}

func errorFromResponse(resp *http.Response) error {

	statusCode := resp.StatusCode
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("%d %s", statusCode, string(body))
}

func writePartField(w *multipart.Writer, fieldname, value, contentType string) error {
	p, err := createFormField(w, fieldname, contentType)
	if err != nil {
		return err
	}
	_, err = p.Write([]byte(value))
	return err
}

// quoteEscaper replaces some special characters in a given string.
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// escapeQuotes replaces single quotes and double-backslashes in the current string.
func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

// createFormField creates the MIME field for a POST request.
func createFormField(w *multipart.Writer, fieldname, contentType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"`, escapeQuotes(fieldname)))
	h.Set("Content-Type", contentType)
	return w.CreatePart(h)
}

func setImpersonationHeaders(req *http.Request, impersonating, sysDN string) {
	// who I want to become
	req.Header.Set("USER_DN", impersonating)
	// who I am
	req.Header.Set("EXTERNAL_SYS_DN", sysDN)
	req.Header.Set("SSL_CLIENT_S_DN", sysDN)
}

// =====================================================================================
// Types copied in from protocol package as of v1.0.18

// Breadcrumb defines a mimimal set of data for clients to display or link to an object's
// parent chain. To get all of a breadcrumb's properties, get the properties associated
// with a breadcrumb's ID.
type Breadcrumb struct {
	ID       string `json:"id"`
	ParentID string `json:"parentId"`
	Name     string `json:"name"`
}

// CallerPermission is a structure defining the attributes for
// permissions granted on an object for the caller of an operation where an
// object is returned
type CallerPermission struct {
	// AllowCreate indicates whether the caller has permission to create child
	// objects beneath this object
	AllowCreate bool `json:"allowCreate"`
	// AllowRead indicates whether the caller has permission to read this
	// object. This is the most fundamental permission granted, and should always
	// be true as only records need to exist where permissions are granted as
	// the system denies access by default. Read access to an object is necessary
	// to perform any other action on the object.
	AllowRead bool `json:"allowRead"`
	// AllowUpdate indicates whether the caller has permission to update this
	// object
	AllowUpdate bool `json:"allowUpdate"`
	// AllowDelete indicates whether the caller has permission to delete this
	// object
	AllowDelete bool `json:"allowDelete"`
	// AllowShare indicates whether the caller has permission to view and
	// alter permissions on this object
	AllowShare bool `json:"allowShare"`
}

// ChangeOwnerRequest is a subset of Object for use to disallow providing certain fields.
type ChangeOwnerRequest struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
	// NewOwner indicates the individual user or group that will become the new
	// owner of the object
	NewOwner string `json:"newOwner"`
	// ApplyRecursively will apply the change owner request to all child objects if true.
	ApplyRecursively bool `json:"applyRecursively"`
}

// ChangeTokenStruct is a nestable structure defining the ChangeToken attribute
// for items in Object Drive
type ChangeTokenStruct struct {
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
}

// CopyObjectRequest is a subset of Object for use to disallow providing certain fields.
type CopyObjectRequest struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
}

// CreateObjectRequest is a subset of Object for use to disallow providing certain fields.
type CreateObjectRequest struct {
	// TypeName reflects the name of the object type associated with TypeID
	TypeName string `json:"typeName"`
	// Name is the given name for the object. (e.g., filename)
	Name string `json:"name"`
	// NamePathDelimiter is an optional delimiter for which the name value is intended
	// to be broken up to create intermediate objects to represent a hierarchy.  by
	// default this value is internally processed as the record separator (char 30).
	// It may be overridden by providing a different string here
	NamePathDelimiter string `json:"namePathDelimiter,omitempty"`
	// Description is an abstract of the object or its contents
	Description string `json:"description"`
	// ParentID can optionally reference another object by its ID in hexadecimal string
	// format to denote the parent/ancestor of this object for hierarchical purposes.
	// Leaving this field empty, or not present indicates that the object should be
	// created at the 'root' or 'top level'
	ParentID string `json:"parentId,omitempty"`
	// RawACM is the raw ACM string that got supplied to create this object
	RawAcm interface{} `json:"acm"`
	// ContentType indicates the mime-type, and potentially the character set
	// encoding for the object contents
	ContentType string `json:"contentType,omitempty"`
	// ContentSize denotes the length of the content stream for this object, in
	// bytes
	ContentSize int64 `json:"contentSize,omitempty"`
	// Permission is the API 1.1+ version for providing permissions for users and groups with a resource and capability driven approach
	Permission Permission `json:"permission,omitempty"`
	// ContainsUSPersonsData indicates if this object contains US Persons data (Yes,No,Unknown)
	ContainsUSPersonsData string `json:"containsUSPersonsData,omitEmpty"`
	// ExemptFromFOIA indicates if this object is exempt from Freedom of Information Act requests (Yes,No,Unknown)
	ExemptFromFOIA string `json:"exemptFromFOIA,omitEmpty"`
	// Properties is an array of Object Properties associated with this object
	Properties []Property `json:"properties,omitempty"`
	// Permissions is the API 1.0 version for providing permissions for users and groups with a share model
	Permissions []ObjectShare `json:"permissions,omitempty"`
	// Owner which could be a group, or different user from the one uploading
	OwnedBy string `json:"ownedBy,omitempty"`
}

// DeleteObjectRequest is a subset of Object for use to disallow providing certain fields.
type DeleteObjectRequest struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
}

// DeletedObjectResponse is the response information provided when an object
// is deleted from Object Drive
type DeletedObjectResponse struct {
	ID string
	// DeletedDate is the timestamp of when an item was deleted.
	DeletedDate time.Time `json:"deletedDate"`
	// CallerPermission is the composite permission the caller has for this object
	CallerPermission CallerPermission `json:"callerPermission,omitempty"`
}

// ExpungedObjectResponse is the response information provided when an object
// is expunged from Object Drive
type ExpungedObjectResponse struct {
	// ExpungedDate is the timestamp of when an item was deleted permanently.
	ExpungedDate time.Time `json:"expungedDate"`
	// CallerPermission is the composite permission the caller has for this object
	CallerPermission CallerPermission `json:"callerPermission,omitempty"`
}

// FilterSetting denotes a field and a condition to match an expression on which to filter results
type FilterSetting struct {
	FilterField string `json:"filterField"`
	Condition   string `json:"condition"`
	Expression  string `json:"expression"`
}

// MoveObjectRequest is a subset of Object for use to disallow providing certain fields.
type MoveObjectRequest struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
	// ParentID is the identifier of the new parent to be assigned, or null if
	// moving to root
	ParentID string `json:"parentId,omitempty"`
}

// Object is a nestable structure defining the base attributes for an Object
// in Object Drive.
type Object struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// CreatedDate is the timestamp of when an item was created.
	CreatedDate time.Time `json:"createdDate"`
	// CreatedBy is the user that created this item.
	CreatedBy string `json:"createdBy"`
	// ModifiedDate is the timestamp of when an item was modified or created.
	ModifiedDate time.Time `json:"modifiedDate"`
	// ModifiedBy is the user that last modified this item
	ModifiedBy string `json:"modifiedBy"`
	// DeletedDate is the timestamp of when an item was deleted
	DeletedDate time.Time `json:"deletedDate"`
	// DeletedBy is the user that last modified this item
	DeletedBy string `json:"deletedBy"`
	// ChangeCount indicates the number of times the item has been modified.
	ChangeCount int `json:"changeCount"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
	// OwnedBy indicates the individual user or group that currently owns the
	// object and has implict full permissions on the object
	OwnedBy string `json:"ownedBy"`
	// TypeID references the ODObjectType by its ID indicating the type of this
	// object
	TypeID string `json:"typeId,omitempty"`
	// TypeName reflects the name of the object type associated with TypeID
	TypeName string `json:"typeName"`
	// Name is the given name for the object. (e.g., filename)
	Name string `json:"name"`
	// Description is an abstract of the object or its contents
	Description string `json:"description"`
	// ParentID references another Object by its ID indicating which object, if
	// any, contains, or is an ancestor of this object. (e.g., folder). An object
	// without a parent is considered to be contained within the 'root' or at the
	// 'top level'.
	ParentID string `json:"parentId,omitempty"`
	// RawACM is the raw ACM string that got supplied to create this object
	RawAcm interface{} `json:"acm"`
	// ContentType indicates the mime-type, and potentially the character set
	// encoding for the object contents
	ContentType string `json:"contentType"`
	// ContentSize denotes the length of the content stream for this object, in
	// bytes
	ContentSize int64 `json:"contentSize"`
	// A sha256 hash of the plaintext as hex encoded string
	ContentHash string `json:"contentHash"`
	// ContainsUSPersonsData indicates if this object contains US Persons data (Yes,No,Unknown)
	ContainsUSPersonsData string `json:"containsUSPersonsData"`
	// ExemptFromFOIA indicates if this object is exempt from Freedom of Information Act requests (Yes,No,Unknown)
	ExemptFromFOIA string `json:"exemptFromFOIA"`
	// Properties is an array of Object Properties associated with this object
	// structured as key/value with portion marking.
	Properties []Property `json:"properties,omitempty"`
	// CallerPermission is the composite permission the caller has for this object
	CallerPermission CallerPermission `json:"callerPermission,omitempty"`
	// Permissions is an array of Object Permissions associated with this object
	// This might be null.  It could have a large list of permission objects
	// relevant to this file (ie: shared with an organization)
	Permissions []Permission_1_0 `json:"permissions,omitempty"`
	// Permission is the API 1.1+ version for providing permissions for users and groups with a resource and capability driven approach
	Permission Permission `json:"permission,omitempty"`
	// Breadcrumbs is an array of Breadcrumb that may be returned on some API calls.
	// Clients can use breadcrumbs to display a list of parents. The top-level
	// parent should be the first item in the slice.
	Breadcrumbs []Breadcrumb `json:"breadcrumbs,omitempty"`
	// IsPDFAvailable is readded back in here to maintain backwards compatibility for
	// integrations that expect this field to exist and will break if it is not present
	IsPDFAvailable bool `json:"isPDFAvailable"`
}

// ObjectShare is the association of an object to one or more users and/or
// groups for specifying permissions that will either be granted or revoked.
// The referenced object is implicit in the URL of the request
type ObjectShare struct {

	// Share indicates the users, groups, or other identities for which the
	// permissions to an object will apply.
	//
	// An ACM compliant share may be expressed as an object. Example format:
	//  "share":{
	//     "users":[
	//        "cn=aldea amanda d cnaldad,ou=people,ou=dia,ou=dod,o=u.s. government,c=us"
	//       ,"cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us"
	//       ]
	//    ,"projects":{
	//        "jifct_twl":{
	//           "disp_nm":"JIFCT.TWL"
	//          ,"groups":[
	//              "SLE"
	//             ,"USER"
	//             ]
	//          }
	//       }
	//    }
	//
	Share interface{} `json:"share,omitEmpty"`

	// AllowCreate indicates whether the users/groups in the share will be
	// granted permission to create child objects beneath the target of this
	// grant when adding permissions or revoking such when removing permissions
	AllowCreate bool `json:"allowCreate,omitEmpty"`

	// AllowRead indicates whether the users/groups in the share will be
	// granted permission to read the object metadata and properties, object
	// stream or list its children when adding permissions or revoking such
	// when removing permissions
	AllowRead bool `json:"allowRead,omitEmpty"`

	// AllowUpdate indicates whether the users/groups in the share will be
	// granted permission to make changes to the object metadata and properties
	// or its content stream when adding permissions or revoking such when
	// removing permissions
	AllowUpdate bool `json:"allowUpdate,omitEmpty"`

	// AllowDelete indicates whether the users/groups in the share will be
	// granted permission to delete the object when adding permissions or
	// revoking such when removing permissions
	AllowDelete bool `json:"allowDelete,omitEmpty"`

	// AllowShare indicates whether the users/groups in the share will be
	// granted permission to share the object to others when adding permissions
	// or revoking such capability when removing permissions
	AllowShare bool `json:"allowShare,omitEmpty"`
}

// ObjectError is a simple list of object identifiers
type ObjectError struct {
	ObjectID string `json:"objectId,omitempty"`
	Error    string `json:"error,omitempty"`
	Msg      string `json:"msg,omitempty"`
	Code     int    `json:"code,omitempty"`
}

// ObjectResultset encapsulates the Object defined herein as an array with
// resultset metric information to expose page size, page number, total rows,
// and page count information when retrieving from the data store
type ObjectResultset struct {
	// Resultset contains meta information about the resultset
	Resultset
	// Objects contains the list of objects in this (page of) results.
	Objects []Object `json:"objects,omitempty"`
	// ObjectErrors is a list of errors per object id
	ObjectErrors []ObjectError `json:"objectErrors,omitempty"`
}

// PagingRequest supports a request constrained to a given page number and size
type PagingRequest struct {
	// PageNumber is the requested page number for this request
	PageNumber int `json:"pageNumber,omitempty"`
	// PageSize is the requested page size for this request
	PageSize int `json:"pageSize,omitempty"`
	// ObjectID if provided provides a focus for paging, often the ParentID
	ObjectID string `json:"objectId,omitempty"`
	// FilterSettings is an array of fitler settings denoting field and conditional match expression to filter results
	FilterSettings []FilterSetting `json:"filterSettings,omitempty"`
	// SortSettings is an array of sort settings denoting a field to sort on and direction
	SortSettings []SortSetting `json:"sortSettings,omitempty"`
	// FilterMatchType indicates the kind of matching performed when multiple filters are provided.
	FilterMatchType string `json:"filterMatchType,omitempty"`
}

// Permission is a nestable structure defining the attributes for
// permissions granted on an object for users who have access to the object
// in Object Drive
type Permission struct {
	// Create contains the resources who are allowed to create child objects
	Create PermissionCapability `json:"create,omitempty"`
	// Read contains the resources who are allowed to read this object metadata,
	// list its contents or view its stream
	Read PermissionCapability `json:"read,omitempty"`
	// Update contains the resources who are allowed to update the object
	// metadata or stream
	Update PermissionCapability `json:"update,omitempty"`
	// Delete contains the resources who are allowed to delete this object,
	// restore it from the trash, or expunge it forever.
	Delete PermissionCapability `json:"delete,omitempty"`
	// Share contains the resources who are allowed to alter the permissions
	// on this object directly, or indirectly via metadata such as acm.
	Share PermissionCapability `json:"share,omitempty"`
}

// Permission_1_0 is a nestable structure defining the attributes for
// permissions granted on an object for users who have access to the object
// in Object Drive
type Permission_1_0 struct {
	Grantee string `json:"grantee,omitempty"`
	// ProjectName contains the project key portion of an AcmShare if this
	// grantee represents a group
	ProjectName string `json:"projectName,omitempty"`
	// ProjectDisplayName contains the disp_nm portion of an AcmShare if this
	// grantee represents a group
	ProjectDisplayName string `json:"projectDisplayName,omitempty"`
	// GroupName contains the group value portion of an AcmShare if this
	// grantee represents a group
	GroupName string `json:"groupName,omitempty"`
	// UserDistinguishedName contains a user value portion of an AcmShare
	// if this grantee represnts a user
	UserDistinguishedName string `json:"userDistinguishedName,omitempty"`
	// DisplayName is a friendly display name suitable for user interfaces for
	// the grantee modeled on the distinguished name common name, or project and group
	DisplayName string `json:"displayName,omitempty"`
	// AllowCreate indicates whether the grantee has permission to create child
	// objects beneath this object
	AllowCreate bool `json:"allowCreate"`
	// AllowRead indicates whether the grantee has permission to read this
	// object. This is the most fundamental permission granted, and should always
	// be true as only records need to exist where permissions are granted as
	// the system denies access by default. Read access to an object is necessary
	// to perform any other action on the object.
	AllowRead bool `json:"allowRead"`
	// AllowUpdate indicates whether the grantee has permission to update this
	// object
	AllowUpdate bool `json:"allowUpdate"`
	// AllowDelete indicates whether the grantee has permission to delete this
	// object
	AllowDelete bool `json:"allowDelete"`
	// AllowShare indicates whether the grantee has permission to view and
	// alter permissions on this object
	AllowShare bool `json:"allowShare"`
}

// PermissionCapability contains the list of resources who are allowed or denied
// the referenced capability
type PermissionCapability struct {
	// AllowedResources is a list of resources who are permitted this capability
	AllowedResources []string `json:"allow,omitempty"`
	// DeniedResources is a list of resources who will be denied this capability
	// even if allowed through other means.
	DeniedResources []string `json:"deny,omitempty"`
}

// Property is a structure defining the attributes for a property
type Property struct {
	// ID is the unique identifier for this property in Object Drive.
	ID string `json:"id"`
	// CreatedDate is the timestamp of when a property was created.
	CreatedDate time.Time `json:"createdDate"`
	// CreatedBy is the user that created this property.
	CreatedBy string `json:"createdBy"`
	// ModifiedDate is the timestamp of when a property was modified or created.
	ModifiedDate time.Time `json:"modifiedDate"`
	// ModifiedBy is the user that last modified this property
	ModifiedBy string `json:"modifiedBy"`
	// ChangeCount indicates the number of times the property has been modified.
	ChangeCount int `json:"changeCount"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken"`
	// Name is the name, key, field, or label given to a property
	Name string `json:"name"`
	// Value is the assigned value for a property.
	Value string `json:"value"`
	// ClassificationPM is the portion mark classification for the value of this
	// property
	ClassificationPM string `json:"classificationPM"`
}

// Resultset provides a summation of an accompanying array of items for which
// it refers from a request with a page number and size.  For example, if the
// request is for page 3 of widgets, with 20 returned per page, and there are 56
// widgets that match, then TotalRows=56, PageCount=3, PageNumber=3, PageSize=20
// and PageRows=16
type Resultset struct {
	// TotalRows is the total number of items matching the same query resulting
	// in this page of results
	TotalRows int `json:"totalRows"`
	// PageCount is the total rows divided by page size
	PageCount int `json:"pageCount"`
	// PageNumber is the requested page number for this resultset
	PageNumber int `json:"pageNumber"`
	// PageSize is the requested page size for this resultset
	PageSize int `json:"pageSize"`
	// PageRows is the number of items included in this page of the results, which
	// may be less than pagesize, but never greater.
	PageRows int `json:"pageRows"`
}

// SortSetting denotes a field and a preferred direction on which to sort results.
type SortSetting struct {
	SortField     string `json:"sortField"`
	SortAscending bool   `json:"sortAscending"`
}

// UpdateObjectAndStreamRequest is a subset of Object for use to disallow providing certain fields.
type UpdateObjectAndStreamRequest struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
	// TypeID references the ODObjectType by its ID indicating the type of this
	// object
	TypeID string `json:"typeId,omitempty"`
	// TypeName reflects the name of the object type associated with TypeID
	TypeName string `json:"typeName"`
	// Name is the given name for the object. (e.g., filename)
	Name string `json:"name"`
	// Description is an abstract of the object or its contents
	Description string `json:"description"`
	// RawACM is the raw ACM string that got supplied to modify this object
	RawAcm interface{} `json:"acm"`
	// Permission is the API 1.1+ version for providing permissions for users and groups with a resource and capability driven approach
	Permission Permission `json:"permission,omitempty"`
	// ContentType indicates the mime-type, and potentially the character set
	// encoding for the object contents
	ContentType string `json:"contentType,omitempty"`
	// ContentSize denotes the length of the content stream for this object, in
	// bytes
	ContentSize int64 `json:"contentSize,omitempty"`
	// ContainsUSPersonsData indicates if this object contains US Persons data (Yes,No,Unknown)
	ContainsUSPersonsData string `json:"containsUSPersonsData,omitEmpty"`
	// ExemptFromFOIA indicates if this object is exempt from Freedom of Information Act requests (Yes,No,Unknown)
	ExemptFromFOIA string `json:"exemptFromFOIA,omitEmpty"`
	// Properties is an array of Object Properties associated with this object
	Properties []Property `json:"properties,omitempty"`
	// RecursiveShare, if true, will apply the updated share permissions to all child objects.
	RecursiveShare bool `json:"recursiveShare"`
}

// UpdateObjectRequest is a subset of Object for use to disallow providing certain fields.
type UpdateObjectRequest struct {
	// ID is the unique identifier for this object in Object Drive.
	ID string `json:"id"`
	// ChangeToken is generated value which is assigned at the database. API calls
	// performing updates must provide the changeToken to be verified against the
	// existing value on record to prevent accidental overwrites.
	ChangeToken string `json:"changeToken,omitempty"`
	// TypeID references the ODObjectType by its ID indicating the type of this
	// object
	TypeID string `json:"typeId,omitempty"`
	// TypeName reflects the name of the object type associated with TypeID
	TypeName string `json:"typeName"`
	// Name is the given name for the object. (e.g., filename)
	Name string `json:"name"`
	// Description is an abstract of the object or its contents
	Description string `json:"description"`
	// RawACM is the raw ACM string that got supplied to modify this object
	RawAcm interface{} `json:"acm"`
	// Permission is the API 1.1+ version for providing permissions for users and groups with a resource and capability driven approach
	Permission Permission `json:"permission,omitempty"`
	// ContentType indicates the mime-type, and potentially the character set
	// encoding for the object contents
	ContentType string `json:"contentType,omitempty"`
	// ContainsUSPersonsData indicates if this object contains US Persons data (Yes,No,Unknown)
	ContainsUSPersonsData string `json:"containsUSPersonsData,omitEmpty"`
	// ExemptFromFOIA indicates if this object is exempt from Freedom of Information Act requests (Yes,No,Unknown)
	ExemptFromFOIA string `json:"exemptFromFOIA,omitEmpty"`
	// Properties is an array of Object Properties associated with this object
	Properties []Property `json:"properties,omitempty"`
	// RecursiveShare, if true, will apply the updated share permissions to all child objects.
	RecursiveShare bool `json:"recursiveShare"`
}
