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
	"testing"
	"time"

	"github.com/karlseguin/ccache"

	"bitbucket.di2e.net/dime/object-drive-server/ciphertext"
	"bitbucket.di2e.net/dime/object-drive-server/client"
	"bitbucket.di2e.net/dime/object-drive-server/config"
	"bitbucket.di2e.net/dime/object-drive-server/dao"
	"bitbucket.di2e.net/dime/object-drive-server/metadata/models"
	"bitbucket.di2e.net/dime/object-drive-server/server"
	"bitbucket.di2e.net/dime/object-drive-server/services/aac"
	"bitbucket.di2e.net/dime/object-drive-server/services/kafka"
	"bitbucket.di2e.net/dime/object-drive-server/util"
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
	clients     []*ClientIdentity
	trafficLogs map[string]*TrafficLog
)

var (
	testIP = flag.String("testIP", "", "The IP address for test API requests. (e.g., docker virtual machine ip)")
)

func TestMain(m *testing.M) {
	// Permit the defer in testMainBody to run its course before exit
	os.Exit(testMainBody(m))
}
func testMainBody(m *testing.M) int {
	flag.Parse()
	testSettings()
	trafficLogs = make(map[string]*TrafficLog)
	trafficLogs[APISampleFile] = NewTrafficLog(APISampleFile)
	defer trafficLogs[APISampleFile].Close()
	// setup populates our global clients
	setup(*testIP)
	if code := stallForAvailability(); code != 0 {
		return code
	}
	code := m.Run()
	cleanupOpenFiles()
	return code
}

func setup(ip string) {
	// We have 11 entries for our clients global var.
	populateClients(11)
}

func cleanupOpenFiles() {
	for i := range clients {
		if clients[i] != nil {
			clients[i].Client.Transport.(*http.Transport).CloseIdleConnections()
		}
	}
}

func testSettings() {
	// Ensure that uses of decryptor will succeed
	os.Setenv(config.OD_TOKENJAR_LOCATION, "../defaultcerts/token.jar")

	root := os.TempDir()
	os.Mkdir(root, 0700)

	key, err := config.MaybeDecrypt(config.GetEnvOrDefault(config.OD_ENCRYPT_MASTERKEY, ""))
	if err != nil {
		fmt.Printf("unable to get encrypt key: %v", err)
		os.Exit(1)
	}

	settings := config.S3CiphertextCacheOpts{
		Root:          root,
		Partition:     "partition0",
		LowWatermark:  .50,
		HighWatermark: .75,
		EvictAge:      300,
		WalkSleep:     30,
		MasterKey:     key,
	}
	zone := ciphertext.S3_DEFAULT_CIPHERTEXT_CACHE
	cache, err2 := ciphertext.NewLocalCiphertextCache(config.RootLogger, zone, settings, "dbID0")
	if err != nil {
		log.Printf("unable to setup ciphertextcache: %v", err2.Error())
	}
	ciphertext.SetCiphertextCache(
		zone,
		cache,
	)
}

func stallForAvailability() int {
	// Don't stall on short tests
	if testing.Short() {
		return 0
	}

	url := mountPoint + "/ping"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("bad request create: %v", err)
		return -11
	}

	// Do this on every try to check the server
	retryFunc := func() int {
		log.Printf("try url: %s", url)
		res, err := clients[0].Client.Do(req)
		if res == nil {
			log.Printf("proxy not ready")
			return -10
		}
		if err != nil {
			log.Printf("bad request: %v", err)
			return -11
		}
		if res.StatusCode != 200 {
			log.Printf("odrive not ready to serve: %d", res.StatusCode)
			return res.StatusCode
		}
		return 0
	}

	// Try every few seconds
	tck := time.NewTicker(1 * time.Second)
	defer tck.Stop()

	// Give up after a while.  We need enough time to cover from when containers are brought up to when they should pass
	timeout := time.After(5 * time.Minute)

	// Attempt to check the server.  Quit if we pass timeout
	for {
		select {
		case <-tck.C:
			code := retryFunc()
			if code == 0 {
				return 0
			}
		case <-timeout:
			return -12
		}
	}
}

func TestTokenJar(t *testing.T) {
	encryptedTest := "ENC{2f2b2f667a7741514944424155474277674a4367734f394465397a49474741466b67793253656b327a66303d}"
	result, err := config.MaybeDecrypt(encryptedTest)
	if err != nil {
		t.Logf("Failed to decrypt value encrypted with token.jar: %v", err)
		t.FailNow()
	}
	if result != "test" {
		t.Logf("Value did not encrypt to the word 'test': %v", err)
		t.FailNow()
	}
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
	C             *client.Client
	Cert          *x509.Certificate
}

func getClientIdentityFromDefaultCerts(component string, certSet string) (*ClientIdentity, error) {
	ci := &ClientIdentity{
		TrustPem: os.ExpandEnv(fmt.Sprintf("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/%s/%s.trust.pem", component, certSet)),
		CertPem:  os.ExpandEnv(fmt.Sprintf("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/%s/%s.cert.pem", component, certSet)),
		KeyPem:   os.ExpandEnv(fmt.Sprintf("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/%s/%s.key.pem", component, certSet)),
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

	// NOTE(cm): We use these paths for old-style test http clients and new client lib Clients.
	trustPath := os.ExpandEnv("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/clients/client.trust.pem")
	certPath := os.ExpandEnv("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/clients/" + name + ".cert.pem")
	keyPath := os.ExpandEnv("$GOPATH/src/bitbucket.di2e.net/dime/object-drive-server/defaultcerts/clients/" + name + ".key.pem")

	ci := &ClientIdentity{
		TrustPem: trustPath,
		CertPem:  certPath,
		KeyPem:   keyPath,
	}
	config, err := newClientTLSConfig(ci)
	if err != nil {
		return nil, fmt.Errorf("cannot get client identity: %v", err)
	}
	ci.Config = config
	ci.Name = name

	// New client.Client instance can be set on field C for use in tests.
	clientConf := client.Config{
		Cert:  certPath,
		Trust: trustPath,
		Key:   keyPath,
		// We expect host to be set globally before this function runs.
		Remote:     mountPoint,
		SkipVerify: true,
	}
	c, err := client.NewClient(clientConf)
	if err != nil {
		return nil, fmt.Errorf("cannot instantiate client.Client: %v", err)
	}
	ci.C = c

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

	cert, err := tls.LoadX509KeyPair(client.CertPem, client.KeyPem)
	if err != nil {
		log.Printf("Error parsing cert: %v", err)
		return nil, err
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("while parsing public certificate from cert and key file %s, %s: %v", client.CertPem, client.KeyPem, err)
	}
	client.Cert = x509Cert

	tlsConfig := &tls.Config{
		InsecureSkipVerify:       true,
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                trustCertPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS10,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

func NewFakeServerWithDAOUsers() *server.AppServer {

	user0, user1, user2 := setupFakeUsers()

	guid, err := util.NewGUID()
	if err != nil {
		log.Printf("Could not create GUID.")
	}
	perms := []models.ODObjectPermission{{Grantee: fakeDN0}}
	perms[0].AllowRead = true
	obj := models.ODObject{Permissions: perms}
	obj.ID = []byte(guid)
	obj.RawAcm.String = server.ValidACMUnclassified
	obj.RawAcm.Valid = true

	fakeDAO := dao.FakeDAO{
		Users:  []models.ODUser{user0, user1, user2},
		Object: obj,
	}

	snippetResponse := aac.SnippetResponse{
		Success:  true,
		Snippets: SnippetTP10,
		Found:    true,
	}
	attributesResponse := aac.UserAttributesResponse{
		Success:        true,
		UserAttributes: "{\"diasUserGroups\":{\"projects\":[{\"projectName\":\"DCTC\",\"groupNames\":[\"ODrive\"]}]}}",
	}
	// Acm needs to have value in f_share that corresponds to the user
	// that is creating objects.  For example, the grantee above for fakeDN0
	// will need cntesttester10oupeopleoudaeouchimeraou_s_governmentcus
	// so that has been put into the ValidAcmUnclassifiedWithFShare value
	acmInfoResponse := aac.AcmInfo{
		Acm:             ValidACMUnclassifiedWithFShare,
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
		UserAttributesResp:         &attributesResponse,
	}

	fakeQueue := kafka.NewFakeAsyncProducer(nil)

	s := server.AppServer{RootDAO: &fakeDAO,
		AAC:           &fakeAAC,
		UsersLruCache: ccache.New(ccache.Configure().MaxSize(1000).ItemsToPrune(50)),
		EventQueue:    fakeQueue,
	}
	// Panics occur if regex routes are not compiled with InitRegex()
	s.InitRegex()
	return &s
}

func setupFakeUsers() (models.ODUser, models.ODUser, models.ODUser) {
	user0 := models.ODUser{DistinguishedName: fakeDN0}
	user1 := models.ODUser{DistinguishedName: fakeDN1}
	user2 := models.ODUser{DistinguishedName: fakeDN2}
	user0.CreatedBy = fakeDN0
	user1.CreatedBy = fakeDN1
	user2.CreatedBy = fakeDN2

	return user0, user1, user2
}
