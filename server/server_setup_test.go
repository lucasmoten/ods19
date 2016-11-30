package server_test

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/karlseguin/ccache"

	"decipher.com/object-drive-server/ciphertext"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/dao"
	"decipher.com/object-drive-server/metadata/models"
	"decipher.com/object-drive-server/server"
	"decipher.com/object-drive-server/services/aac"
	"decipher.com/object-drive-server/services/kafka"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

var (
	fakeDN0 = `cn=test tester10,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`
	fakeDN1 = `cn=test tester01,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`
	fakeDN2 = `cn=test tester02,ou=people,ou=dae,ou=chimera,o=u.s. government,c=us`
)

const (
	APISampleFile = "APISample"
)

var (
	host        string
	clients     []*ClientIdentity
	trafficLogs map[string]*TrafficLog
)

func setup(ip string) {

	if ip == "" {
		host = fmt.Sprintf("https://%s:%s", config.DockerVM, config.Port)
	} else {
		host = fmt.Sprintf("https://%s:%s", ip, config.Port)
	}

	if !testing.Short() {
		generatePopulation()
	}
}

var (
	testIP              = flag.String("testIP", "", "The IP address for test API requests. Usually the dockerVM")
	dumpFileDescriptors = flag.Bool("dumpFileDescriptors", false, "Set to true to shell out to lsof before, during, and after tests")
)

func countOpenFiles() int {
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("lsof -p %v", os.Getpid())).Output()
	if err != nil {
		log.Printf("no lsof on this machine: %v", err)
		return 0
	}
	log.Print(string(out))
	lines := strings.Split(string(out), "\n")
	return len(lines) - 1
}

func dumpOpenFiles(shouldPrint bool, at string) {
	if shouldPrint {
		fmt.Printf("filehandles at %s: %d\n", at, countOpenFiles())
	}
}

func cleanupOpenFiles() {
	for i := range clients {
		if clients[i] != nil {
			clients[i].Client.Transport.(*http.Transport).CloseIdleConnections()
		}
	}
}

func testSettings() {
	root := os.TempDir()
	os.Mkdir(root, 0700)
	settings := config.S3CiphertextCacheOpts{
		Root:          root,
		Partition:     "partition0",
		LowWatermark:  .50,
		HighWatermark: .75,
		EvictAge:      300,
		WalkSleep:     30,
		MasterKey:     config.GetEnvOrDefault(config.OD_ENCRYPT_MASTERKEY, ""),
	}
	zone := ciphertext.S3_DEFAULT_CIPHERTEXT_CACHE
	cache, err := ciphertext.NewLocalCiphertextCache(config.RootLogger, zone, settings, "dbID0")
	if err != nil {
		log.Printf("unable to setup ciphertextcache: %v", err.Error())
	}
	ciphertext.SetCiphertextCache(
		zone,
		cache,
	)
}

func testMainBody(m *testing.M) int {
	flag.Parse()

	testSettings()
	trafficLogs = make(map[string]*TrafficLog)
	trafficLogs[APISampleFile] = NewTrafficLog(APISampleFile)
	dumpOpenFiles(*dumpFileDescriptors, "TestMain before setup")
	setup(*testIP)
	dumpOpenFiles(*dumpFileDescriptors, "TestMain after setup")
	code := m.Run()
	trafficLogs[APISampleFile].Close()
	dumpOpenFiles(*dumpFileDescriptors, "TestMain after run")
	cleanupOpenFiles()
	dumpOpenFiles(*dumpFileDescriptors, "TestMain after cleanup")
	return code
}

func TestMain(m *testing.M) {
	os.Exit(testMainBody(m))
}

func generatePopulation() {
	//We have 11 test certs (note the test_0 is known as tester10, and the last is twl-server-generic)
	population := 11
	populateClients(population)
}

func populateClients(population int) {
	clients = make([]*ClientIdentity, population)
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
	config, err := newClientTLSConfig(ci)
	if err != nil {
		log.Printf("Cannot get identity: %v", err)
		return nil, err
	}
	ci.Config = config
	ci.Name = fmt.Sprintf("%s_%s", component, certSet)
	return ci, nil
}

func getClientIdentity(i int, name string) (*ClientIdentity, error) {
	ci := &ClientIdentity{
		TrustPem: os.ExpandEnv("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/clients/client.trust.pem"),
		CertPem:  os.ExpandEnv("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/clients/" + name + ".cert.pem"),
		KeyPem:   os.ExpandEnv("$GOPATH/src/decipher.com/object-drive-server/defaultcerts/clients/" + name + ".key.pem"),
	}
	config, err := newClientTLSConfig(ci)
	if err != nil {
		log.Printf("Cannot get identity: %v", err)
		return nil, err
	}
	ci.Config = config
	ci.Name = name

	return ci, nil
}

// newClientTLSConfig creates a per-client tls config
func newClientTLSConfig(client *ClientIdentity) (*tls.Config, error) {

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

func NewFakeServerWithDAOUsers() *server.AppServer {

	user1, user2 := setupFakeUsers()

	guid, err := util.NewGUID()
	if err != nil {
		log.Printf("Could not create GUID.")
	}
	perms := []models.ODObjectPermission{{Grantee: fakeDN1}}
	perms[0].AllowRead = true
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
	// Acm needs to have value in f_share that corresponds to the user
	// that is creating objects.  For example, the grantee above for fakeDN1
	// will need cntesttester01oupeopleoudaeouchimeraou_s_governmentcus
	// so that has been put into the ValidAcmUnclassifiedWithFShare value
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

	fakeQueue := kafka.NewFakeAsyncProducer(nil)

	s := server.AppServer{RootDAO: &fakeDAO,
		ServicePrefix: config.RootURLRegex,
		AAC:           &fakeAAC,
		UsersLruCache: ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50)),
		EventQueue:    fakeQueue,
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
