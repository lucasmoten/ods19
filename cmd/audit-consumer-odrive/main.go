package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/Shopify/sarama"
	"github.com/samuel/go-thrift/thrift"
	"github.com/spacemonkeygo/openssl"
	"github.com/tidwall/gjson"

	"decipher.com/object-drive-server/legacyssl"

	auditsvc "github.com/deciphernow/gm-fabric-go/audit/audittransformationservice_thrift"
	auditevent "github.com/deciphernow/gm-fabric-go/audit/events_thrift"
)

var (
	conf = flag.String("conf", "config.json", "path to json config")
)

func main() {
	flag.Parse()
	config, err := NewAuditConsumerConfig(*conf)
	if err != nil {
		log.Fatal(err)
	}

	ac, err := NewAuditConsumer(config)
	if err != nil {
		log.Fatal(err)
	}
	err = ac.Start()
	if err != nil {
		log.Fatalf("could not start application: %v", err)
	}

	done := make(chan bool)
	<-done

}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// NewAuditConsumerConfig reads a json configuration from disk.
func NewAuditConsumerConfig(path string) (AuditConsumerConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return AuditConsumerConfig{}, err
	}
	var conf AuditConsumerConfig
	err = json.Unmarshal(data, &conf)
	if err != nil {
		return AuditConsumerConfig{}, err
	}
	return conf, nil
}

// AuditConsumerConfig holds data required to set up the audit service.
type AuditConsumerConfig struct {
	KafkaAddrs []string `json:"kafka_addrs"`
	ZKAddrs    []string `json:"zk_addrs"`
	KafkaTopic string   `json:"kafka_topic"`
	// Direct connect to Audit Transformation Svc
	AuditSvcIP   string `json:"audit_svc_ip"`
	AuditSvcPort string `json:"audit_svc_port"`
	// Discover Audit Transformation Svc
	AuditZKPath string `json:"audit_zk_path"`
	Trust       string `json:"trust"`
	Cert        string `json:"cert"`
	Key         string `json:"key"`
	// Defaults are the json-configured defaults
	Defaults map[string]string
}

// AuditConsumer drains a kafka queue and forwards data to the Audit Service.
type AuditConsumer struct {
	consumer sarama.Consumer
	config   AuditConsumerConfig
	errs     chan error
	svc      *auditsvc.AuditTransformationServiceClient
}

// NewAuditConsumer creates an AuditConsumer from a config.
func NewAuditConsumer(conf AuditConsumerConfig) (*AuditConsumer, error) {

	// Feature flag. Hide until we're talking to Audit Service.
	//if false {
	dialOpts := &legacyssl.OpenSSLDialOptions{}
	dialOpts.SetInsecureSkipHostVerification()
	conn, err := legacyssl.NewOpenSSLConn(
		conf.Trust, conf.Cert, conf.Key, conf.AuditSvcIP, conf.AuditSvcPort, dialOpts)
	if err != nil {
		return nil, err
	}
	svc := NewThriftClient(conn)
	_ = svc
	//} // end feature flag

	// Kafka consumer
	c, err := sarama.NewConsumer(conf.KafkaAddrs, nil)
	if err != nil {
		return nil, err
	}

	ac := AuditConsumer{config: conf, svc: svc, consumer: c}

	return &ac, nil
}

// Start starts the consumer. Connections are established in the constructor.
func (ac *AuditConsumer) Start() error {
	topics, err := ac.consumer.Topics()
	if err != nil {
		return err
	}
	if !nameFound(ac.config.KafkaTopic, topics) {
		return fmt.Errorf("topic %s not found", ac.config.KafkaTopic)
	}

	partitions, err := ac.consumer.Partitions(ac.config.KafkaTopic)
	if err != nil {
		return err
	}

	pc, err := ac.consumer.ConsumePartition(ac.config.KafkaTopic, partitions[0], 0)
	if err != nil {
		return err
	}

	log.Println("offset:", pc.HighWaterMarkOffset())

	done := make(chan bool)

	go func() {
		fmt.Println("Start consuming events.")
		for event := range pc.Messages() {
			ae, err := unmarshal(event.Value)
			if err != nil {
				//ac.errs <- err
				log.Println("could not unmarshal audit:", err)
				continue
			}
			// TODO should buffer these and submit as a batch
			submit := []*auditevent.AuditEvent{&ae}
			resp, err := ac.svc.SubmitAuditEvents(submit)
			if err != nil {
				// Error messages are useless, for now. Logs, less so. ssh to box:
				// tail -f /opt/bedrock/audit-transformation-service/logs/application.log
				log.Println(err, resp)
			}
			if resp != nil {
				log.Println("\taudit resp status:", resp.Status)
				log.Println("\taudit resp messages:", resp.Messages)
				log.Println("\taudit resp code:", resp.StatusCode)

			}

		}
		done <- true
	}()

	<-done
	return nil
}

// Stop initiates the stop routine.
func (ac *AuditConsumer) Stop() {}

// unmarshal creates an AuditEvent from inner json with gjson library. We do
// not need to know the app-specific shape of the "payload" field, only that
// the field is there.
func unmarshal(data []byte) (auditevent.AuditEvent, error) {
	var ret auditevent.AuditEvent
	// Access the audit_event field, a sub-field of payload.
	result := gjson.GetBytes(data, "payload.audit_event")
	err := json.Unmarshal([]byte(result.Raw), &ret)
	if err != nil {
		return ret, errors.New("could not unmarshal audit_event")
	}
	return ret, nil
}

func nameFound(name string, names []string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// NewThriftClient embeds an openssl conn in the audit service client.
func NewThriftClient(conn *openssl.Conn) *auditsvc.AuditTransformationServiceClient {
	trns := thrift.NewTransport(thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
	thriftClient := thrift.NewClient(trns, true)
	c := auditsvc.AuditTransformationServiceClient{Client: thriftClient}
	return &c
}
