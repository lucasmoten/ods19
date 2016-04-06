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

	"decipher.com/object-drive-server/config"
	"decipher.com/object-drive-server/services/audit"
	"decipher.com/object-drive-server/services/audit/generated/acm_thrift"
	auditservice "decipher.com/object-drive-server/services/audit/generated/auditservice_thrift"
	"decipher.com/object-drive-server/services/audit/generated/components_thrift"
	auditevents "decipher.com/object-drive-server/services/audit/generated/events_thrift"
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

	//trustPath := filepath.Join(config.CertsDir, "server", "server.trust.pem")
	//certPath := filepath.Join(config.CertsDir, "server", "server.cert.pem")
	//keyPath := filepath.Join(config.CertsDir, "server", "server.key.pem")
	//opts := &config.OpenSSLDialOptions{}
	//opts.SetInsecureSkipHostVerification()
	//host := "10.2.11.46"
	//port := "10443"
	//client, err := audit.NewRESTAuditor(trustPath, certPath, keyPath, host, port, opts)
	//if err != nil {
	//	t.Errorf("Could not create RESTAuditClient: %v", err)
	//}

	//	f, _ := ioutil.ReadFile("minimal.json")
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

func TestEventAccess(t *testing.T) {

	if !*useVPN {
		t.Skipf(skipTestMsg)
	}

	e := getMinimalEventAccess()
	data := []*auditevents.AuditEvent{e}

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

func getMinimalEventAccess() *auditevents.AuditEvent {
	e := &auditevents.AuditEvent{}
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

func stringPtr(s string) *string { return &s }
func int32Ptr(i int32) *int32    { return &i }
func boolPtr(b bool) *bool       { return &b }
