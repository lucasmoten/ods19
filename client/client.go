package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"decipher.com/object-drive-server/protocol"
)

// ObjectDrive defines operations for our client (and eventually our server).
type ObjectDrive interface {
	GetObject(id string) (protocol.Object, error)
	CreateObject(protocol.CreateObjectRequest, io.Reader) (protocol.Object, error)
	GetObjectStream(id string) (io.Reader, error)
	DeleteObject(id string, token string) (protocol.DeletedObjectResponse, error)
}

// Client implents ObjectDrive.
type Client struct {
	httpClient *http.Client
	url        string
}

// Verify that Client Implements ObjectDrive.
var _ ObjectDrive = (*Client)(nil)

// Config defines the bare minimum that must be statically configured for a Client.
type Config struct {
	Cert   string
	Trust  string
	Key    string
	Remote string
}

// Opt modifies a client passed to it.
type Opt func(*Client) *Client

// NewClient instantiates a new Client that implements ObjectDrive.  This client can be used to perform
// CRUD operations on a running ObjectDrive instance.
//
// The client requires a configuration structure that contains the key bits of information necessary to
// establish a connection to the ObjectDrive: certificates, trusts, keys, and remote URL.
func NewClient(conf Config, opts ...Opt) (*Client, error) {
	trust, err := ioutil.ReadFile(conf.Trust)
	if err != nil {
		return nil, err
	}
	caPool := x509.NewCertPool()
	if caPool.AppendCertsFromPEM(trust) == false {
		return nil, err
	}
	cert, err := tls.LoadX509KeyPair(conf.Cert, conf.Key)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify:       true,
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                caPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS10,
	}
	tlsConfig.BuildNameToCertificate()

	var c http.Client
	c.Transport = &http.Transport{TLSClientConfig: tlsConfig}

	return &Client{&c, conf.Remote}, nil
}

// CreateObject performs the create operation on the ObjectDrive from the CreateObjectRequest that fully
// specifies the object to be created.  The caller must also provide an open io.Reader interface to the stream
// they wish to upload.  If creating an object with no filestream, such as a folder, then reader must be nil.
func (c *Client) CreateObject(obj protocol.CreateObjectRequest, reader io.Reader) (protocol.Object, error) {
	putURL := c.url + "/objects"
	var newObj protocol.Object

	jsonBody, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return newObj, err
	}

	body := bytes.Buffer{}
	contentType := ""

	// If an io.Reader is passed, then the data will be uploaded.  Otherwise, only metadata will be
	// uploaded with no associated filestream
	if reader != nil {
		writer := multipart.NewWriter(&body)

		writePartField(writer, "ObjectMetadata", string(jsonBody), "application/json")
		part, err := writer.CreateFormFile("filestream", obj.Name)
		if err != nil {
			return newObj, err
		}

		if _, err = io.Copy(part, reader); err != nil {
			return newObj, err
		}

		err = writer.Close()
		if err != nil {
			return newObj, err
		}

		contentType = writer.FormDataContentType()
	} else {
		body.Write([]byte(jsonBody))
	}

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", putURL, &body)
	if err != nil {
		return newObj, err
	}

	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", "application/json")
	// Only set for filestreams
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Submit the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return newObj, err
	}

	defer resp.Body.Close()

	// Send back the created object properties
	err = json.NewDecoder(resp.Body).Decode(&newObj)
	if err != nil {
		return newObj, err
	}

	return newObj, nil
}

// GetObject returns an the metadata associated with an object based on it's unique ID.  This metadata
// can be used to facilitate further operations and modifications on the object.
func (c *Client) GetObject(id string) (protocol.Object, error) {
	var obj protocol.Object

	propertyURL := c.url + "/objects/" + id + "/properties"

	meta, err := c.httpClient.Get(propertyURL)
	if err != nil {
		return obj, err
	}

	body, err := ioutil.ReadAll(meta.Body)
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

	resp, err := c.httpClient.Get(fileURL)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// DeleteObject moves an object on the server to the trash.  The object's ID and changetoken from the
// current object in ObjectDrive are needed to perform the operation.
func (c *Client) DeleteObject(id string, token string) (protocol.DeletedObjectResponse, error) {
	url := c.url + "/objects/" + id + "/trash"
	var deleteResponse protocol.DeletedObjectResponse
	var deleteRequest = protocol.DeleteObjectRequest{
		ID:          id,
		ChangeToken: token,
	}

	jsonBody, err := json.MarshalIndent(deleteRequest, "", "    ")
	if err != nil {
		return deleteResponse, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return deleteResponse, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Submit the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return deleteResponse, err
	}

	defer resp.Body.Close()

	// Send back the created object properties
	err = json.NewDecoder(resp.Body).Decode(&deleteResponse)
	if err != nil {
		return deleteResponse, err
	}

	return deleteResponse, nil

}

// writePartField
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
