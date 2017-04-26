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
	"os"
	"strings"

	"decipher.com/object-drive-server/protocol"
)

// ObjectDrive defines operations for our client (and eventually our server).
type ObjectDrive interface {
	GetObject(id string) (protocol.Object, error)
	CreateObject(protocol.CreateObjectRequest, io.Reader) (protocol.Object, error)
	GetObjectStream(id string) (io.Reader, error)
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
func NewClient(conf Config, opts ...Opt) (*Client, error) {
	log.Printf("Starting client with Cert: %s \n", conf.Cert)

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

// CreateObject performs the create operation on the ObjectDrive; writing a local file up to the drive.
func (c *Client) CreateObject(obj protocol.CreateObjectRequest, reader io.Reader) (protocol.Object, error) {
	putURL := c.url + "/objects"
	var newObj protocol.Object

	log.Printf("Starting to upload something to %s", putURL)

	f, err := os.Open(obj.Name)
	if err != nil {
		return newObj, err
	}

	body := bytes.Buffer{}
	writer := multipart.NewWriter(&body)

	jsonBody, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return newObj, err
	}

	writePartField(writer, "ObjectMetadata", string(jsonBody), "application/json")
	part, err := writer.CreateFormFile("filestream", obj.Name)
	if err != nil {
		return newObj, err
	}

	if _, err = io.Copy(part, f); err != nil {
		return newObj, err
	}

	err = writer.Close()
	if err != nil {
		return newObj, err
	}

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", putURL, &body)
	if err != nil {
		return newObj, err
	}

	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())

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

// GetObject returns an object's metadata properties.
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

// WriteObject retrieves an object and writes it to the filesystem.
func WriteObject(name string, reader io.Reader) error {
	file, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(name, file, os.FileMode(int(0700)))
	if err != nil {
		return err
	}

	return nil
}

func writePartField(w *multipart.Writer, fieldname, value, contentType string) error {
	p, err := createFormField(w, fieldname, contentType)
	if err != nil {
		return err
	}
	_, err = p.Write([]byte(value))
	return err
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func createFormField(w *multipart.Writer, fieldname, contentType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"`, escapeQuotes(fieldname)))
	h.Set("Content-Type", contentType)
	return w.CreatePart(h)
}
