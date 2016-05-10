package audit_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"testing"
	"time"

	"decipher.com/object-drive-server/cmd/metadataconnector/libs/mapping"
	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/services/audit/generated/acm_thrift"
	auditservice "decipher.com/object-drive-server/services/audit/generated/auditservice_thrift"
	"decipher.com/object-drive-server/services/audit/generated/components_thrift"
	"decipher.com/object-drive-server/services/audit/generated/events_thrift"
	"decipher.com/object-drive-server/util"
	"decipher.com/object-drive-server/util/testhelpers"
)

var thriftAuditClient *audit.ThriftAuditClient
var restAuditClient *audit.RESTAuditClient

var (
	useVPN          = flag.Bool("useVPN", false, "Pass -useVPN=true to connect to Audit Service test instance")
	auditHost       = flag.String("auditHost", "10.2.11.46", "The IP Address of Audit Service test instance")
	auditThriftPort = flag.String("auditPort", "9050", "The Audit Service thrift port")
	auditRESTPort   = flag.String("auditRESTPort", "10443", "The Audit Service https port")
)

var (
	skipTestMsg = "Skipping test because -useVPN=true was not passed to `go test`"
	thriftPort  string
	restPort    string
	host        string
)

func setupDefaults() {
	host = *auditHost
	thriftPort = *auditThriftPort
	restPort = *auditRESTPort
}

func setupConnections() {

	trustPath := filepath.Join(config.CertsDir, "server", "server.trust.pem")
	certPath := filepath.Join(config.CertsDir, "server", "server.cert.pem")
	keyPath := filepath.Join(config.CertsDir, "server", "server.key.pem")

	opts := &config.OpenSSLDialOptions{}
	opts.SetInsecureSkipHostVerification()
	conn, err := config.NewOpenSSLTransport(
		trustPath, certPath, keyPath, host, thriftPort, opts)
	if err != nil {
		log.Fatal("Could not connect to Audit service in test")
	}

	thriftAuditClient = audit.NewThriftAuditor(conn)
	thriftAuditClient.Start()

	restAuditClient, _ = audit.NewRESTAuditor(trustPath, certPath, keyPath, host, restPort, opts)
	restAuditClient.Start()
}

func TestMain(m *testing.M) {
	flag.Parse()
	setupDefaults()
	if *useVPN {
		setupConnections()
	}

	m.Run()
}

func TestRESTEventAccess(t *testing.T) {

	t.Skipf("Trying to avoid REST")

	e := getMinimalEventAccess()

	marshaled, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		t.Errorf("Could not marshal event into JSON: %v", err)
	}

	uri := "https://" + host + ":" + restPort + "/EventAccesses"
	req, _ := http.NewRequest("POST", uri, bytes.NewBuffer(marshaled))
	req.Header.Set("Content-Type", "application/json")

	if testing.Verbose() {
		fmt.Println("DUMPING REQUEST")
		dumped, _ := httputil.DumpRequest(req, true)
		fmt.Println(string(dumped))
	}

	resp, err := restAuditClient.Client.Do(req)
	if err != nil {
		t.Errorf("Error making HTTP request: %v", err)
	}

	var auditResp auditservice.AuditResponse
	buf, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(buf, &auditResp)
	if err != nil {
		t.Errorf("Could not unmarshal AuditResponse: %v", err)
	}

	if resp.StatusCode != 200 {
		fmt.Println("Got this response without code 200")
		fmt.Printf("Status received %v\n", auditResp.Status)
		fmt.Printf("Messages received %v\n", auditResp.Messages)
		t.Fail()
	}

	if resp.StatusCode == 200 {
		fmt.Println("SUCCESSFUL RESTFUL REQUEST TO AUDITSVC")
		fmt.Println(resp)
	}

}

func TestEventAccesses(t *testing.T) {
	var event events_thrift.AuditEvent
	_ = event

	preHandlerEventFields(&event, t)
}
func TestEventCreates(t *testing.T) {
}
func TestEventDeletes(t *testing.T)   {}
func TestEventModifies(t *testing.T)  {}
func TestEventSearchQry(t *testing.T) {}

func preHandlerEventFields(event *events_thrift.AuditEvent, t *testing.T) {
	myIP := "192.168.11.100"

	rawAcm := testhelpers.ValidACMTopSecretSITK

	convertedAcm, err := mapping.RawAcmToThriftAcm(rawAcm)
	if err != nil {
		t.Errorf("preHandlerEventFields failed: could not convert rawAcm: %v\n", err)
	}

	audit.WithActionInitiator(
		event, "DISTINGUISHED_NAME", "CN=Holmes Jonathan,OU=People,OU=Bedrock,OU=Six 3 Systems,O=U.S. Government,C=US")
	audit.WithNTPInfo(event, "IP_ADDRESS", "2016-03-16T19:14:50.164Z", "1.2.3.4")
	audit.WithActionMode(event, "USER_INITIATED")
	audit.WithActionLocations(event, "IP_ADDRESS", myIP)
	audit.WithActionTarget(event, "IP_ADDRESS", myIP, convertedAcm)
	audit.WithActionTargetVersions(event, "1.0")
	audit.WithSessionIds(event, newSessionID())
	audit.WithCreator(event, "APPLICATION", "Object Drive")

}

func TestEventAccess(t *testing.T) {

	if !*useVPN {
		t.Skipf(skipTestMsg)
	}

	e := getMinimalEventAccess()
	data := []*events_thrift.AuditEvent{e}

	for _, e := range data {
		thriftAuditClient.Log(e)
	}

	time.Sleep(7000 * time.Millisecond)
	fmt.Println("Sleep over")

}

func TestBlackHoleAuditor(t *testing.T) {

	bha := audit.NewBlackHoleAuditor()
	bha.Start()
	e := getMinimalEventAccess()
	bha.Log(e)
	t.Log("Event pass async.")

}

func getMinimalEventAccess() *events_thrift.AuditEvent {
	e := &events_thrift.AuditEvent{}
	e.Type = stringPtr("EventAccess")
	e.Action = stringPtr("ACCESS")
	e.ActionInitiator = &components_thrift.ActionInitiator{
		IdentityType: stringPtr("DISTINGUISHED_NAME"),
		Value:        stringPtr("CN=User Test, OU=People, OU=Orion, OU=HumanGeo, O=U.S. Government, C=US"),
	}
	e.ActionLocations = []*components_thrift.ActionLocation{
		{
			Identifier: stringPtr("IP_ADDRESS"),
			Value:      stringPtr("192.168.3.4"),
		},
	}
	e.ActionMode = stringPtr("USER_INITIATED")
	e.ActionResult = stringPtr("SUCCESS")
	e.ActionTargetMessages = []string{"successful access", "totally works"}
	e.ActionTargetVersions = []string{"2.4"}
	// ActionTargets are the actual objects?

	testAcm := &acm_thrift.Acm{
		Version:      stringPtr("2.1.0"),
		Classif:      "TS",
		OwnerProd:    []string{"USA"},
		DissemCtrls:  []string{"NF"},
		ExFromRollup: stringPtr("true"),
		Portion:      stringPtr("TS"),
		Banner:       stringPtr("TOP SECRET"),
	}

	e.ActionTargets = []*components_thrift.ActionTarget{
		{
			IdentityType: stringPtr("FULLY_QUALIFIED_DOMAIN_NAME"),
			Value:        stringPtr("https://bedrock.363-283.io/bedrock/"),
			Acm:          testAcm,
		},
	}

	e.CreatedOn = stringPtr("2016-03-16T19:15:06.831Z")
	e.Creator = &components_thrift.Creator{
		IdentityType: stringPtr("APPLICATION"),
		Value:        stringPtr("Bedrock Shoebox"),
	}
	e.Edh = &components_thrift.Edh{
		Guide: &components_thrift.Guide{Prefix: stringPtr("9999")},
		ResponsibleEntity: &components_thrift.ResponsibleEntity{
			Country:         stringPtr("USA"),
			Organization:    stringPtr("DIA"),
			SubOrganization: stringPtr("DCTC"),
		},
		Security: &components_thrift.Security{
			OwnerProducer:        stringPtr("USA"),
			ClassifiedBy:         stringPtr("9999"),
			ClassificationReason: stringPtr("Illuminati"),
			DeclassDate:          stringPtr("2018-01-01"),
			DerivedFrom:          stringPtr("Multiple Sources"),
		},
	}
	e.NtpInfo = &components_thrift.NTPInfo{
		IdentityType: stringPtr("IP_ADDRESS"),
		LastUpdate:   stringPtr("2016-03-16T19:14:50.164Z"),
		Server:       stringPtr("1.2.3.4"),
	}
	e.Resources = []*components_thrift.Resource{
		{
			ObjectType: stringPtr("Resource"),
			Name: &components_thrift.ResourceName{
				Title: stringPtr("Random.docx"),
				Acm:   testAcm,
			},
			Location:           stringPtr("https://bedrock.363-283.io/bedrock/#!/journal/56e1d644e4b00330bbf354e1"),
			Size:               int32Ptr(32),
			SubType:            stringPtr("shoebox comment"),
			Type:               stringPtr("OBJECT"),
			Role:               stringPtr("GENERAL_USER"),
			MalwareCheck:       boolPtr(true),
			MalwareCheckStatus: stringPtr("SUCCESS"),
			Content: &components_thrift.ResourceContent{
				Send: boolPtr(false),
			},
			Description: &components_thrift.ResourceDescription{
				Description: stringPtr("This is a description of the document"),
				Acm:         testAcm,
			},
			MalwareServices: []*components_thrift.MalwareService{
				&components_thrift.MalwareService{Service: stringPtr("Norton")},
			},
			Identifier: stringPtr("bedrock::shoebox.document:56e1d644e4b00330bbf354e2"),
			Parent: &components_thrift.ResourceParent{
				Type:       stringPtr("DOCUMENT"),
				SubType:    stringPtr("shoebox document"),
				Location:   stringPtr("https://bedrock.363-283.io/bedrock/#!/journal/56e1d644e4b00330bbf354e1"),
				Identifier: stringPtr("bedrock::shoebox.document:56e1d644e4b00330bbf354e1"),
			},
			Acm: testAcm,
		},
	}
	e.SessionIds = []string{"24gfsrg35gfg3g34"}
	e.Workflow = &components_thrift.Workflow{
		//Complete: boolPtr(true),
		Id: stringPtr("Stage 3"),
	}

	return e
}

// "A little copying is better than the wrong abstraction." - Rob Pike

func stringPtr(s string) *string { return &s }
func int32Ptr(i int32) *int32    { return &i }
func boolPtr(b bool) *bool       { return &b }

func newSessionID() string {
	id, err := util.NewGUID()
	if err != nil {
		return "unknown"
	}
	return id
}
