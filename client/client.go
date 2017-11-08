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
	"strings"

	"decipher.com/object-drive-server/protocol"
	"github.com/deciphernow/gm-fabric-go/tlsutil2"
)

// ObjectDrive defines operations for our client (and eventually our server).
type ObjectDrive interface {
	CreateObject(protocol.CreateObjectRequest, io.Reader) (protocol.Object, error)
	ChangeOwner(protocol.ChangeOwnerRequest) (protocol.Object, error)
	DeleteObject(id string, token string) (protocol.DeletedObjectResponse, error)
	GetObject(id string) (protocol.Object, error)
	GetObjectStream(id string) (io.Reader, error)
	MoveObject(protocol.MoveObjectRequest) (protocol.Object, error)
	UpdateObject(protocol.UpdateObjectRequest) (protocol.Object, error)
	UpdateObjectAndStream(protocol.UpdateObjectAndStreamRequest, io.Reader) (protocol.Object, error)
}

// Client implements ObjectDrive.
type Client struct {
	httpClient *http.Client
	url        string
	// Verbose will print extra debug information if true.
	Verbose bool
	conf    Config
	MyDN    string
}

// Verify that Client Implements ObjectDrive.
var _ ObjectDrive = (*Client)(nil)

// Config defines the bare minimum that must be statically configured for a Client.
type Config struct {
	Cert       string
	Trust      string
	Key        string
	SkipVerify bool
	// Remote specifies the full API proxy prefix: https://{host}:{port}/{prefix}
	// Actual object drive API endpoints are appended to this string.
	Remote string
	// Impersonation is a DN of a user we want to impersonate. If set, HTTP headers
	// USER_DN will be set to this value, and EXTERNAL_SYS_DN and SSL_CLIENT_S_DN
	// will be set to the Client.MyDN field.
	Impersonation string
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
		return nil, err
	}
	cert, err := tls.LoadX509KeyPair(conf.Cert, conf.Key)
	if err != nil {
		return nil, fmt.Errorf("while opening cert and key file %s, %s: %v", conf.Cert, conf.Key, err)
	}

	pub, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("while parsing public certificate from cert and key file %s, %s: %v", conf.Cert, conf.Key, err)
	}
	mydn := tlsutil2.GetDistinguishedName(pub)

	tlsConfig := &tls.Config{
		InsecureSkipVerify:       conf.SkipVerify,
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                caPool,
		RootCAs:                  caPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS10,
	}
	tlsConfig.BuildNameToCertificate()

	var c http.Client
	c.Transport = &http.Transport{TLSClientConfig: tlsConfig}

	return &Client{&c, conf.Remote, false, conf, mydn}, nil
}

// CreateObject performs the create operation on the ObjectDrive from the CreateObjectRequest that fully
// specifies the object to be created.  The caller must also provide an open io.Reader interface to the stream
// they wish to upload.  If creating an object with no filestream, such as a folder, then reader must be nil.
func (c *Client) CreateObject(req protocol.CreateObjectRequest, reader io.Reader) (protocol.Object, error) {
	uri := c.url + "/objects"
	var ret protocol.Object

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
		part, err := writer.CreateFormFile("filestream", req.Name)
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

	if c.conf.Impersonation != "" {
		setImpersonationHeaders(httpReq, c.conf.Impersonation, c.MyDN)
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

	// Send back the created object properties
	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

// GetObject fetches the metadata associated with an object by its unique ID.
func (c *Client) GetObject(id string) (protocol.Object, error) {
	var obj protocol.Object

	propertyURL := c.url + "/objects/" + id + "/properties"

	req, err := http.NewRequest("GET", propertyURL, nil)
	if err != nil {
		return obj, err
	}

	if c.conf.Impersonation != "" {
		setImpersonationHeaders(req, c.conf.Impersonation, c.MyDN)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return obj, err
	}

	if resp.StatusCode != 200 {
		return obj, fmt.Errorf("got HTTP error code: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return obj, err
	}

	jsonErr := json.Unmarshal(body, &obj)
	if jsonErr != nil {
		return obj, jsonErr
	}

	return obj, nil
}

// GetObjectStream fetches the filestream associated with an object, if any exists.
func (c *Client) GetObjectStream(id string) (io.Reader, error) {
	fileURL := c.url + "/objects/" + id + "/stream"

	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		return nil, err
	}

	if c.conf.Impersonation != "" {
		setImpersonationHeaders(req, c.conf.Impersonation, c.MyDN)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return nil, fmt.Errorf("http status: %v", resp.StatusCode)
	}

	return resp.Body, nil
}

// DeleteObject moves an object on the server to the trash.  The object's ID and changetoken from the
// current object in ObjectDrive are needed to perform the operation.
func (c *Client) DeleteObject(id string, token string) (protocol.DeletedObjectResponse, error) {

	url := c.url + "/objects/" + id + "/trash"

	var deleteResponse protocol.DeletedObjectResponse

	deleteRequest := protocol.DeleteObjectRequest{
		ID:          id,
		ChangeToken: token,
	}

	resp, err := c.doPost(url, deleteRequest)
	if err != nil {
		log.Println(err)
		return deleteResponse, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&deleteResponse)
	if err != nil {
		return deleteResponse, err
	}

	return deleteResponse, nil
}

// ChangeOwner changes the object's ownedBy field.
func (c *Client) ChangeOwner(req protocol.ChangeOwnerRequest) (protocol.Object, error) {
	uri := c.url + "/objects/" + req.ID + "/owner/" + req.NewOwner
	var ret protocol.Object

	resp, err := c.doPost(uri, req)
	if err != nil {
		return ret, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, fmt.Errorf("could not decode response: %v", err)
	}

	return ret, nil
}

// MoveObject moves a given file or folder into a new parent folder, both specified by ID.
func (c *Client) MoveObject(req protocol.MoveObjectRequest) (protocol.Object, error) {
	uri := c.url + "/objects/" + req.ID + "/move/" + req.ParentID
	var ret protocol.Object

	resp, err := c.doPost(uri, req)
	if err != nil {
		return ret, fmt.Errorf("error performing request: %v", err)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, fmt.Errorf("could not decode response: %v", err)
	}

	return ret, nil
}

// UpdateObject updates an object's metadata or permissions. To update an actual
// filestream, use UpdateObjectAndStream.
func (c *Client) UpdateObject(req protocol.UpdateObjectRequest) (protocol.Object, error) {
	uri := c.url + "/objects/" + req.ID + "/properties"
	var ret protocol.Object

	resp, err := c.doPost(uri, req)
	if err != nil {
		return ret, fmt.Errorf("http error %v: %v", resp.StatusCode, err)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, fmt.Errorf("could not decode response: %v", err)
	}

	return ret, nil
}

// UpdateObjectAndStream updates an object's associated stream as well as its metadata or permissions.
func (c *Client) UpdateObjectAndStream(req protocol.UpdateObjectAndStreamRequest, r io.Reader) (protocol.Object, error) {
	uri := c.url + "/objects/" + req.ID + "/stream"
	var ret protocol.Object

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

	if c.conf.Impersonation != "" {
		setImpersonationHeaders(httpReq, c.conf.Impersonation, c.MyDN)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.Println(err)
		return ret, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		return ret, fmt.Errorf("could not decode response: %v", err)
	}

	return ret, nil
}

func (c *Client) doPost(uri string, body interface{}) (*http.Response, error) {
	jsonBody, err := json.MarshalIndent(body, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("could not marshall json body: %v", err)
	}

	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	if c.conf.Impersonation != "" {
		setImpersonationHeaders(req, c.conf.Impersonation, c.MyDN)
	}

	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
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

func setImpersonationHeaders(req *http.Request, impersonating, sysDNs string) {
	// who I want to become
	req.Header.Set("USER_DN", impersonating)
	// who I am
	req.Header.Set("EXTERNAL_SYS_DN", sysDNs)
	req.Header.Set("SSL_CLIENT_S_DN", sysDNs)

}
