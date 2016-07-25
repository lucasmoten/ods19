package server_test

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"decipher.com/object-drive-server/cmd/odrive/libs/dao"
	"decipher.com/object-drive-server/cmd/odrive/libs/server"
	cfg "decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/protocol"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

var (
	fakeDN0 = `cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`
	fakeDN1 = `cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`
	fakeDN2 = `cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`
)

var (
	host    string
	clients []*ClientIdentity
)

func setup(ip string) {

	if ip == "" {
		host = fmt.Sprintf("https://%s:%s", cfg.DockerVM, cfg.Port)
	} else {
		host = fmt.Sprintf("https://%s:%s", ip, cfg.Port)
	}

	log.Println("Using this host address for server_test:", host)
	if !testing.Short() {
		generatePopulation()
	}
}

var testIP = flag.String("testIP", "", "The IP address for test API requests. Usually the dockerVM")

func TestMain(m *testing.M) {
	flag.Parse()
	setup(*testIP)
	code := m.Run()
	os.Exit(code)
}

func generatePopulation() {
	//We have 11 test certs (note the test_0 is known as tester10, and the last is twl-server-generic)
	population := 11
	populateClients(population)
}

func populateClients(population int) {
	clients = make([]*ClientIdentity, population)
	usersReq, _ := http.NewRequest("GET", host+cfg.NginxRootURL+"/users", nil)
	for i := 0; i < len(clients); i++ {
		var clientname string

		// Construct clients
		switch i {
		case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9:
			clientname = fmt.Sprintf("test_%d", i)
			client, err := getClientIdentity(i, clientname)
			if err != nil {
				log.Printf("Could not create client %d: %v", i, err)
			}
			clients[i] = client
		case 10:
			client, err := getClientIdentityFromDefaultCerts("server", "server")
			if err != nil {
				log.Printf("Could not create client for server/server: %v", err)
			}
			clients[i] = client
		default:
			log.Fatalf("Aborting test setup. Unknown client id: %d", i)
		}

		transport := &http.Transport{TLSClientConfig: clients[i].Config}
		clients[i].Client = &http.Client{Transport: transport}

		// force creation of the user in the database
		resp, err := clients[i].Client.Do(usersReq)
		if err != nil {
			log.Printf("Error in populateClients: %v/n", err)
		}
		ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()

		// TODO assign groups in another switch
		switch i {
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
	Client        *http.Client
	Groups        []string
}

func getClientIdentityFromDefaultCerts(component string, certSet string) (*ClientIdentity, error) {
	ci := &ClientIdentity{
		TrustPem: os.ExpandEnv(fmt.Sprintf("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/%s/%s.trust.pem", component, certSet)),
		CertPem:  os.ExpandEnv(fmt.Sprintf("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/%s/%s.cert.pem", component, certSet)),
		KeyPem:   os.ExpandEnv(fmt.Sprintf("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/%s/%s.key.pem", component, certSet)),
	}
	cfg, err := NewClientTLSConfig(ci)
	if err != nil {
		log.Printf("Cannot get identity: %v", err)
		return nil, err
	}
	ci.Config = cfg
	ci.Name = fmt.Sprintf("%s_%s", component, certSet)
	return ci, nil
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

func makeFolderViaJSON(folderName string, clientid int, t *testing.T) *protocol.Object {

	nameWithTimestamp := folderName + strconv.FormatInt(time.Now().Unix(), 10)

	obj, err := makeFolderWithACMViaJSON(nameWithTimestamp, testhelpers.ValidACMUnclassified, clientid)
	if err != nil {
		t.Errorf("Error creating folder %s: %v\n", folderName, err)
		t.FailNow()
	}
	return obj
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

	snippetResponse := aac.SnippetResponse{
		Success:  true,
		Snippets: testhelpers.SnippetTP10,
	}
	acmInfoResponse := aac.AcmInfo{
		Acm:             testhelpers.ValidACMUnclassifiedWithFShare,
		IncludeInRollup: false,
	}
	acmResponse := aac.AcmResponse{
		Success:   true,
		Messages:  []string{"FakeAAC AcmResponse"},
		AcmValid:  true,
		HasAccess: true,
		AcmInfo:   &acmInfoResponse,
	}
	checkAccessResponse := aac.CheckAccessResponse{
		Success:   true,
		Messages:  []string{"FakeAAC CheckAccessResponse"},
		HasAccess: true,
	}
	var acmResponseArray []*aac.AcmResponse
	acmResponseArray = append(acmResponseArray, &acmResponse)
	checkAccessAndPopulateResponse := aac.CheckAccessAndPopulateResponse{
		Success:         true,
		Messages:        []string{"FakeAAC CheckAccessAndPopulateResponse"},
		AcmResponseList: acmResponseArray,
	}
	// Fake the AAC interface
	fakeAAC := aac.FakeAAC{
		ACMResp:                    &acmResponse,
		CheckAccessResp:            &checkAccessResponse,
		CheckAccessAndPopulateResp: &checkAccessAndPopulateResponse,
		SnippetResp:                &snippetResponse,
	}

	s := server.AppServer{RootDAO: &fakeDAO,
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

func makeUserShare(userDN string) interface{} {
	shareString := fmt.Sprintf(`{"users":["%s"]}`, userDN)
	var shareInterface interface{}
	json.Unmarshal([]byte(shareString), &shareInterface)
	return shareInterface
}

func makeGroupShare(project string, displayName string, groupName string) interface{} {
	shareString := fmt.Sprintf(`{"projects":{"%s":{"disp_nm":"%s","groups":["%s"]}}}`, project, displayName, groupName)
	var shareInterface interface{}
	json.Unmarshal([]byte(shareString), &shareInterface)
	return shareInterface
}
