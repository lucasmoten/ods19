package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"decipher.com/object-drive-server/config"
	auditservice "decipher.com/object-drive-server/services/audit/generated/auditservice_thrift"
	// TODO remove this alias
	auditevents "decipher.com/object-drive-server/services/audit/generated/events_thrift"
	"github.com/samuel/go-thrift/thrift"
	"github.com/spacemonkeygo/openssl"
)

// HandlerFunc ...
type HandlerFunc func(events chan *auditevents.AuditEvent, payloads *Payloads)

// Auditor is a generic interface that can wrap an auditing framework or
// logging framework.
type Auditor interface {
	Log(event interface{})
	Start()
}

// BlackHoleAuditor is an Auditor implementation that does not require the
// network. Log messages are thrown away. Useful for testing.
type BlackHoleAuditor struct {
	events       chan *auditevents.AuditEvent
	PayloadQueue *Payloads
	Logged       []*auditevents.AuditEvent
}

// ThriftAuditClient sends AuditEvent messages to the Audit Service via Thrift.
type ThriftAuditClient struct {
	events       chan *auditevents.AuditEvent
	PayloadQueue *Payloads
	Svc          auditservice.AuditService
}

// RESTAuditClient sends AuditEvent messages to the Audit Service via HTTPS.
type RESTAuditClient struct {
	events       chan *auditevents.AuditEvent
	PayloadQueue *Payloads
	Client       *http.Client
}

// NewThriftAuditor creates a ThriftAuditClient.
func NewThriftAuditor(conn *openssl.Conn) *ThriftAuditClient {

	// TODO should this return an error?
	eventsChan := make(chan *auditevents.AuditEvent, 100)

	trns := thrift.NewTransport(
		thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	thriftClient := thrift.NewClient(trns, true)
	svc := auditservice.AuditServiceClient{Client: thriftClient}
	payloads := NewPayloads(DefaultMaxRequestArraySize)
	c := &ThriftAuditClient{
		Svc:          &svc,
		events:       eventsChan,
		PayloadQueue: payloads,
	}

	return c
}

// NewRESTAuditor creates a RESTAuditClient.
func NewRESTAuditor(trustPath, certPath, keyPath, host, port string, dialOpts *config.OpenSSLDialOptions) (*RESTAuditClient, error) {
	httpClient, err := config.NewOpenSSLHTTPClient(trustPath, certPath, keyPath, host, port, dialOpts)
	if err != nil {
		return nil, fmt.Errorf("Could not create an http.Client: %v", err)
	}

	eventsChan := make(chan *auditevents.AuditEvent, 100)
	payloads := NewPayloads(DefaultMaxRequestArraySize)
	rc := &RESTAuditClient{
		events:       eventsChan,
		Client:       httpClient,
		PayloadQueue: payloads,
	}
	return rc, nil
}

// NewBlackHoleAuditor creates an Auditor interface that does not communicate
// externally.
func NewBlackHoleAuditor() *BlackHoleAuditor {

	eventsChan := make(chan *auditevents.AuditEvent, 100)
	payloads := NewPayloads(DefaultMaxRequestArraySize)
	logged := make([]*auditevents.AuditEvent, 0)

	return &BlackHoleAuditor{
		events:       eventsChan,
		PayloadQueue: payloads,
		Logged:       logged,
	}
}

// Start a RESTAuditClient.
func (c *RESTAuditClient) Start() {
	go handleAuditEvents(c.events, c.PayloadQueue)
	go c.handleDoRequest()
}

// Start a ThriftAuditClient.
func (c *ThriftAuditClient) Start() {
	go handleAuditEvents(c.events, c.PayloadQueue)
	go c.handleDoRequest()
}

// Start a BlackHoleAuditor.
func (c *BlackHoleAuditor) Start() {
	// go handleAuditEvents(c.events, c.PayloadQueue)
	// go c.blackHole(c.PayloadQueue)
}

// Log satisfies the Auditor interface.
func (c *RESTAuditClient) Log(event interface{}) {
	put(event, c.events)
}

// Log satisfies the Auditor interface.
func (c *ThriftAuditClient) Log(event interface{}) {
	put(event, c.events)
}

// Log satisfies the Auditor interface.
func (c *BlackHoleAuditor) Log(event interface{}) {
	// Do not log to channel to keep BlackHoleAuditor non-async.
	e, ok := event.(*auditevents.AuditEvent)
	if !ok {
		log.Println("Invalid event passed to Log")
		return
	}
	if e.Type == nil {
		log.Println("You must provide an event type")
		return
	}
	c.PayloadQueue.Add(*e.Type, e)
	c.Logged = append(c.Logged, e)
}

func put(event interface{}, events chan *auditevents.AuditEvent) {
	e, ok := event.(*auditevents.AuditEvent)
	if !ok {
		log.Println("Invalid event type passed to Log")
		return
	}
	events <- e
}

func (c *RESTAuditClient) handleDoRequest() {
	// TODO ...

}

// handleAuditEvents reads from the an events chan and places the audit events on a queue.
func handleAuditEvents(events chan *auditevents.AuditEvent, payloads *Payloads) {

	for e := range events {
		fmt.Println("GOT AN EVENT")
		// other prerequisites?
		if e.Type == nil {
			log.Println("You must provide the Type field")
			continue
		}

		payloads.Add(*e.Type, e)
	}
}

// getMaxPayloadsFromQueue extracts a slice of events to send to the Audit service
// from a queue inside the Payloads struct. Only the queue referenced by the
// string key is read from. Callers should test for a 0 length slice.
func getMaxPayloadsFromQueue(key string, payloads *Payloads) []*auditevents.AuditEvent {

	payload := make([]*auditevents.AuditEvent, 0)
	q, ok := payloads.M[key]
	if !ok {
		log.Println("Invalid key for Payload.M:", key)
	}

	// loop until max payload size, or until nothing left in queue
	for i := 0; i <= payloads.RequestArraySize; i++ {
		if q.count <= 0 {
			break
		}
		event := q.Pop()
		if event == nil {
			break
		} else {
			payload = append(payload, event.Value)
		}
	}

	return payload
}

// blackHole throws away the payload.
func (c *BlackHoleAuditor) blackHole(payloads *Payloads) {
	for {
		for key := range payloads.M {
			payload := getMaxPayloadsFromQueue(key, payloads)
			msg := "BlackHoleAuditor: logged payload length %v for type %s\n"
			if len(payload) > 0 {
				fmt.Printf(msg, len(payload), key)
				c.Logged = append(c.Logged, payload...)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *ThriftAuditClient) handleDoRequest() {

	fmt.Println("STARTING DOREQUEST HANDLER")

	for {
		// Iterate the keys in map[string]*Queue.
		for key := range c.PayloadQueue.M {

			// For each key, create a payload slice.
			payload := getMaxPayloadsFromQueue(key, c.PayloadQueue)

			if len(payload) > 0 {
				// TODO: what if this request fails?
				resp, err := c.doThriftRequest(key, payload)
				if err != nil {
					goto ErrorRoutine
				}
				if resp == nil {
					log.Println("Calling doRequest returned nil response")
				}
				if resp != nil {
					log.Printf("GOT RESPONSE. STATUS: %v\n", resp.Status)
					fmt.Println(resp.Messages)
				}
				if resp.Status == "SUCCESS" {
					log.Printf("Request successful. Messages: %v\n", resp.Messages)
				}
				if resp.Status == "MALFORMED" {
					log.Printf("Request was malformed. Reason: %v\n", resp.Messages)
				}
				if resp.Status == "FAIL" {
					log.Printf("Request failed. Reason: %v\n", resp.Messages)
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

ErrorRoutine:
	log.Println("ERROR ROUTINE.")
	// TODO ...

}

// doThriftRequest switches on the string event type passed in for each audit event
// and calls the appropriate api method.
func (c *ThriftAuditClient) doThriftRequest(eventType string, payload []*auditevents.AuditEvent) (*auditservice.AuditResponse, error) {

	switch eventType {
	case "EventAccess":
		log.Printf("Got event type: %s\n", eventType)
		return c.Svc.SubmitAuditEvents(payload)
	case "EventAuthenticate":
		log.Printf("Got event type: %s\n", eventType)
		return c.Svc.SubmitEventAuthenticates(payload)
	case "EventCreate":
		log.Printf("Got event type: %s\n", eventType)
		return c.Svc.SubmitEventCreates(payload)
	case "EventDelete":
		log.Printf("Got event type: %s\n", eventType)
		return c.Svc.SubmitEventDeletes(payload)
	case "EventExport":
		log.Printf("Got event type: %s\n", eventType)
		return c.Svc.SubmitEventExports(payload)
	case "EventImport":
		log.Printf("Got event type: %s\n", eventType)
		return c.Svc.SubmitEventImports(payload)
	case "EventModify":
		log.Printf("Got event type: %s\n", eventType)
		return c.Svc.SubmitEventModifies(payload)
	case "EventSearchQry":
		log.Printf("Got event type: %s\n", eventType)
		return c.Svc.SubmitEventSearchQrys(payload)
	case "EventSystemAction":
		log.Printf("Got event type: %s\n", eventType)
		return c.Svc.SubmitEventSystemActions(payload)
	case "EventUnknown":
		return c.Svc.SubmitEventUnknowns(payload)
	default:
		return nil, fmt.Errorf("Unknown event type: %s\n", eventType)
	}
}

// doPost is a helper function that wraps execution of an HTTP POST with a JSON
// payload of any type.
func doPost(client *http.Client, payload interface{}, uri string) (*http.Response, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(b))
	dump, _ := httputil.DumpRequest(req, true)
	fmt.Printf("%s\n", dump)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func debugJSON(event *auditevents.AuditEvent) {
	e, err := json.MarshalIndent(event, "", "   ")
	if err != nil {
		log.Println("Could not marshal AuditEvent to JSON!")
	}
	fmt.Print(string(e))
}
