package server_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/dao"
	"decipher.com/object-drive-server/cmd/metadataconnector/libs/server"
	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

//var fakeDN1 = `CN=test tester01, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US`
//var fakeDN2 = `CN=test tester02, O=U.S. Government, OU=chimera, OU=DAE, OU=People, C=US`
//var fakeDN1 = `cn=test tester01,o=u.s. government,ou=chimera,ou=dae,ou=people,c=us`
//var fakeDN2 = `cn=test tester02,o=u.s. government,ou=chimera,ou=dae,ou=people,c=us`
var fakeDN1 = `cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`
var fakeDN2 = `cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`

var host string
var clients []*ClientIdentity
var httpclients []*http.Client

func init() {
	host = fmt.Sprintf("https://%s:%s", cfg.DockerVM, cfg.Port)
	log.Println("Using this host address for server_test:", host)
	generatePopulation()

}

func generatePopulation() {
	//We have 10 test certs (note the test_0 is known as tester10)
	population := 10
	populateClients(population)
}

func populateClients(population int) {
	clients = make([]*ClientIdentity, population)
	httpclients = make([]*http.Client, population)
	usersReq, _ := http.NewRequest("GET", host+cfg.NginxRootURL+"/users", nil)
	for i := 0; i < len(clients); i++ {
		client, err := getClientIdentity(i, "test_"+strconv.Itoa(i))
		clients[i] = client
		if err != nil {
			log.Printf("Could not create client %d: %v", i, err)
		} else {
			//log.Printf("Creating client %d", i)
		}

		transport := &http.Transport{TLSClientConfig: clients[i].Config}
		httpclients[i] = &http.Client{Transport: transport}
		// Fire-and-Forget call to /users which will force creation of the
		// user in the database
		_, err = httpclients[i].Do(usersReq)
		if err != nil {
			log.Printf("Error in populateClients: %v/n", err)
		}
	}
}

// ClientIdentity is a user that is going to connect to our service
type ClientIdentity struct {
	TrustPem      string
	CertPem       string
	KeyPem        string
	Config        *tls.Config
	Name          string
	UploadCache   string
	DownloadCache string
	Index         int
}

func getClientIdentity(i int, name string) (*ClientIdentity, error) {
	ci := &ClientIdentity{
		TrustPem: os.ExpandEnv("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/clients/client.trust.pem"),
		CertPem:  os.ExpandEnv("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/clients/" + name + ".cert.pem"),
		KeyPem:   os.ExpandEnv("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/clients/" + name + ".key.pem"),
	}
	cfg, err := NewClientTLSConfig(ci)
	if err != nil {
		log.Printf("Cannot get identity: %v", err)
		return nil, err
	}
	ci.Config = cfg
	ci.Name = name

	//Keep this huge directory out of $GOPATH
	if os.ExpandEnv("$AUTOPILOT_HOME") == "" {
		os.Setenv("$AUTOPILOT_HOME", os.ExpandEnv("$HOME/autopilot"))
		os.Mkdir("~/autopilot", 0700)
	}
	ci.UploadCache = os.ExpandEnv("$HOME/autopilot/uploadCache" + name)
	ci.DownloadCache = os.ExpandEnv("$HOME/autopilot/downloadCache" + name)
	ci.Index = i
	_, err = os.Stat(ci.UploadCache)
	if os.IsNotExist(err) {
		err = os.MkdirAll(ci.UploadCache, 0700)
		if err != nil {
			log.Printf("Unable to make an upload cache for %s:%v", ci.UploadCache, err)
			return nil, err
		}
	}
	_, err = os.Stat(ci.DownloadCache)
	if os.IsNotExist(err) {
		err = os.MkdirAll(ci.DownloadCache, 0700)
		if err != nil {
			log.Printf("Unable to make a download cache for %s:%v", name, err)
			return nil, err
		}
	}
	return ci, nil
}

// NewClientTLSConfig creates a per-client tls config
func NewClientTLSConfig(client *ClientIdentity) (*tls.Config, error) {

	// Create the trust
	trustBytes, err := ioutil.ReadFile(client.TrustPem)
	if err != nil {
		log.Printf("Unable to read %s: %v", client.TrustPem, err)
		return nil, err
	}
	trustCertPool := x509.NewCertPool()
	if trustCertPool.AppendCertsFromPEM(trustBytes) == false {
		log.Printf("Error parsing cert: %v", err)
		return nil, err
	}

	//Create certkeypair
	cert, err := tls.LoadX509KeyPair(client.CertPem, client.KeyPem)
	if err != nil {
		log.Printf("Error parsing cert: %v", err)
		return nil, err
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
		// Ensure that we only use our "CA" to validate certificates
		ClientCAs: trustCertPool,
		// PFS because we can but this will reject client with RSA certificates
		// CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		// Force it server side
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS10,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

func makeFolderViaJSON(folderName string, clientid int) (*protocol.Object, error) {
	return makeFolderWithACMViaJSON(folderName, testhelpers.ValidACMUnclassified, clientid)
}
func makeFolderWithACMViaJSON(folderName string, rawAcm string, clientid int) (*protocol.Object, error) {
	folderuri := host + cfg.NginxRootURL + "/objects"
	folder := protocol.Object{}
	folder.Name = folderName
	folder.TypeName = "Folder"
	folder.ParentID = ""
	folder.RawAcm = rawAcm
	jsonBody, err := json.Marshal(folder)
	if err != nil {
		log.Printf("Unable to marshal json for request:%v", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", folderuri, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Error setting up HTTP Request: %v", err)
		return nil, err
	}
	// do the request
	req.Header.Set("Content-Type", "application/json")
	transport := &http.Transport{TLSClientConfig: clients[clientid].Config}
	client := &http.Client{Transport: transport}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to do request:%v", err)
		return nil, err
	}
	// process Response
	if res.StatusCode != http.StatusOK {
		log.Printf("bad status: %s", res.Status)
		return nil, errors.New("Status was " + res.Status)
	}
	var createdFolder protocol.Object
	err = util.FullDecode(res.Body, &createdFolder)
	if err != nil {
		log.Printf("Error decoding json to Object: %v", err)
		log.Println()
		return nil, err
	}
	return &createdFolder, nil
}

func NewFakeServerWithDAOUsers() *server.AppServer {

	user1, user2 := setupFakeUsers()

	userCache := server.NewUserCache()
	snippetCache := server.NewSnippetCache()

	guid, err := util.NewGUID()
	if err != nil {
		log.Printf("Could not create GUID.")
	}
	perms := []models.ODObjectPermission{
		{Grantee: fakeDN1, AllowRead: true}}
	obj := models.ODObject{Permissions: perms}
	obj.ID = []byte(guid)
	obj.RawAcm.String = testhelpers.ValidACMUnclassified
	obj.RawAcm.Valid = true

	fakeDAO := dao.FakeDAO{
		Users:  []models.ODUser{user1, user2},
		Object: obj,
	}

	checkAccessResponse := aac.CheckAccessResponse{
		Success:   true,
		HasAccess: true,
	}
	// Fake the AAC interface
	fakeAAC := aac.FakeAAC{
		CheckAccessResp: &checkAccessResponse,
	}

	s := server.AppServer{DAO: &fakeDAO,
		ServicePrefix: cfg.RootURLRegex,
		AAC:           &fakeAAC,
		Users:         userCache,
		Snippets:      snippetCache,
	}
	// Panics occur if regex routes are not compiled with InitRegex()
	s.InitRegex()
	return &s
}

func setupFakeUsers() (models.ODUser, models.ODUser) {
	user1 := models.ODUser{DistinguishedName: fakeDN1}
	user2 := models.ODUser{DistinguishedName: fakeDN2}
	user1.CreatedBy = fakeDN1
	user2.CreatedBy = fakeDN2

	return user1, user2
}
