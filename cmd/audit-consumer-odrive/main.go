package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"decipher.com/object-drive-server/legacyssl"

	"github.com/Shopify/sarama"
	"github.com/samuel/go-thrift/thrift"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/spacemonkeygo/openssl"
	"github.com/tidwall/gjson"

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

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1)
	<-sigchan
	fmt.Println("Shutting down.")
	ac.Stop()
	fmt.Println("Exited cleanly.")
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

// AuditConsumer drains a kafka queue and forwards data to the Audit Service.
type AuditConsumer struct {
	consumer  *Kafka
	config    AuditConsumerConfig
	errs      chan error
	svc       *Audit
	zkConn    *ZK
	messages  chan *sarama.ConsumerMessage
	quitChans []chan bool
}

// NewAuditConsumer creates an AuditConsumer from a config. Mutexes are initialized
// for protecting internal connections shared between goroutines.
func NewAuditConsumer(conf AuditConsumerConfig) (*AuditConsumer, error) {
	ac := AuditConsumer{
		config:   conf,
		svc:      &Audit{m: &sync.RWMutex{}},
		consumer: &Kafka{m: &sync.RWMutex{}},
		zkConn:   &ZK{m: &sync.RWMutex{}},
	}
	return &ac, nil
}

// Start starts the consumer. Connections are established in the constructor.
func (ac *AuditConsumer) Start() error {
	zkQuit, err := ac.zkLoop()
	if err != nil {
		return fmt.Errorf("error from zkLoop: %v", err)
	}
	ac.registerQuitChan(zkQuit)

	kafkaQuit, err := ac.kafkaLoop()
	if err != nil {
		return fmt.Errorf("error from kafkaLoop: %v", err)
	}
	ac.registerQuitChan(kafkaQuit)

	auditQuit, err := ac.auditLoop()
	if err != nil {
		return fmt.Errorf("error from auditLoop: %v", err)
	}
	ac.registerQuitChan(auditQuit)

	topics, err := ac.consumer.Consumer().Topics()
	if err != nil {
		return err
	}
	if !nameFound(ac.config.KafkaTopic, topics) {
		return fmt.Errorf("topic %s not found", ac.config.KafkaTopic)
	}

	partitions, err := ac.consumer.Consumer().Partitions(ac.config.KafkaTopic)
	if err != nil {
		return err
	}

	pc, err := ac.consumer.Consumer().ConsumePartition(ac.config.KafkaTopic, partitions[0], sarama.OffsetNewest)
	if err != nil {
		return err
	}

	log.Println("offset:", pc.HighWaterMarkOffset())

	// Message consumption routine. When this cancels, write an offset
	// TODO: make this restartable
	// TODO: commit offset to disk
	go func() {
		fmt.Println("Start consuming events.")
		for event := range pc.Messages() {
			ae, err := unmarshal(event.Value)
			if err != nil {
				//ac.errs <- err
				log.Println("could not unmarshal audit:", err)
				continue
			}
			ae = ApplyAuditDefaults(ae, ac.config)
			// TODO should buffer these and submit as a batch
			submit := []*auditevent.AuditEvent{&ae}
			resp, err := ac.svc.Conn().SubmitAuditEvents(submit)
			if err != nil {
				// ssh to box:
				// tail -f /opt/bedrock/audit-transformation-service/logs/application.log
				log.Println(err, resp)
				ac.svc.reconnect = true
			}
			if resp != nil {
				log.Println("\taudit resp status:", resp.Status)
				log.Println("\taudit resp messages:", resp.Messages)
				log.Println("\taudit resp code:", resp.StatusCode)
			}
		}

	}()

	return nil
}

// Stop initiates the stop routine.
func (ac *AuditConsumer) Stop() {
	for _, quit := range ac.quitChans {
		quit <- true
	}
}

func (ac *AuditConsumer) auditLoop() (chan bool, error) {

	host, port, err := auditNode(ac.config, ac.zkConn)
	if err != nil {
		return nil, err
	}
	log.Printf("connecting to audit-transformation-service %s:%s\n", host, port)
	dialOpts := &legacyssl.OpenSSLDialOptions{}
	dialOpts.SetInsecureSkipHostVerification()
	conn, err := legacyssl.NewOpenSSLConn(ac.config.Trust, ac.config.Cert, ac.config.Key, host, port, dialOpts)
	if err != nil {
		return nil, err
	}
	svc := NewThriftClient(conn)
	if svc == nil {
		return nil, errors.New("could not make initial connection to audit service")
	}
	ac.svc.SetConn(svc)

	quit := make(chan bool)
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if ac.svc.reconnect {
					host, port, err = auditNode(ac.config, ac.zkConn)
					if err != nil {
						log.Println("ERROR: attempted reconnect: could not locate audit service address")
						continue
					}
					conn, err := legacyssl.NewOpenSSLConn(ac.config.Trust, ac.config.Cert, ac.config.Key, host, port, dialOpts)
					if err != nil {
						log.Println("ERROR: attempted reconnect: could not open ssl conn to audit service")
						continue
					}
					svc := NewThriftClient(conn)
					ac.svc.SetConn(svc)
					ac.svc.reconnect = false
				}
			case <-quit:
				return
			}
		}

	}()

	return nil, nil
}

func (ac *AuditConsumer) kafkaLoop() (chan bool, error) {

	quit := make(chan bool)

	// anonymous function that selects our brokers
	brokers := func(conf AuditConsumerConfig, z *ZK) []string {
		if len(conf.KafkaAddrs) < 1 && len(conf.ZKAddrs) > 0 {
			return buildBrokers(z.Conn(), "/brokers/ids")
		}
		return ac.config.KafkaAddrs
	}

	// initial connection to kafka
	c, err := sarama.NewConsumer(brokers(ac.config, ac.zkConn), nil)
	if err != nil {
		return nil, err
	}
	ac.consumer.SetConsumer(c)

	ticker := time.NewTicker(30 * time.Second)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, err := ac.consumer.Consumer().Topics()
				if err != nil {
					log.Println("health check failed: attempting kafka reconnect")
					c, err := sarama.NewConsumer(brokers(ac.config, ac.zkConn), nil)
					if err != nil {
						continue
					}
					ac.consumer.SetConsumer(c)
				}
			case <-quit:
				err = ac.consumer.Consumer().Close()
				if err != nil {
					log.Printf("error closing consumer: %v", err)
				}
				return
			}
		}
	}()

	return quit, nil
}

func (ac *AuditConsumer) registerQuitChan(ch chan bool) {
	if ac.quitChans == nil {
		ac.quitChans = make([]chan bool, 0)
	}
	if ch == nil {
		// no-op
		return
	}
	ac.quitChans = append(ac.quitChans, ch)
}

// zkLoop starts a routine that will keep a connection to zookeeper alive
func (ac *AuditConsumer) zkLoop() (chan bool, error) {

	if len(ac.config.ZKAddrs) < 1 {
		log.Println("no zookeeper configured")
		return nil, nil
	}

	conn, _, err := zk.Connect(ac.config.ZKAddrs, time.Second*5)
	if err != nil {
		return nil, err
	}
	ac.zkConn.SetConn(conn)

	quit := make(chan bool)

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				state := ac.zkConn.Conn().State().String()
				switch state {
				case "StateConnecting", "StateConnected", "StateHasSession":
					continue
				default:
					log.Printf("attempting zk reconnect due to state: %s\n", state)
					conn, _, err := zk.Connect(ac.config.ZKAddrs, time.Second*5)
					if err != nil {
						log.Printf("zk reconnect failure: %v\n", err)
						continue
					}
					ac.zkConn.SetConn(conn)
				}
			case <-quit:
				ac.zkConn.Conn().Close()
				return
			}
		}
	}()

	return quit, nil
}

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

// buildBrokers queries a zookeeper path and returns a string slice of host:port pairs
// suitable for the kafka library's constructor. Errors are ignored, because the caller
// can decide what to do if a zero-length list of brokers is returned.
// TODO: this should go in a shared kafka library. Kafka paths in ZK are highly standardized.
func buildBrokers(conn *zk.Conn, path string) []string {

	var brokers []string
	children, _, _ := conn.Children(path)
	for _, c := range children {
		data, _, err := conn.Get(path + "/" + c)
		if err != nil {
			break
		}
		type addr struct {
			Host string
			Port int
		}
		var a addr
		if err := json.Unmarshal(data, &a); err != nil {
			break
		}
		brokers = append(brokers, fmt.Sprintf("%s:%v", a.Host, a.Port))
	}
	return brokers

}

// auditNode discovers the audit service and returns host, port, and a possible error.
// If IP and port are configured directly, use those values. Otherwise, use Zookeeper.
func auditNode(conf AuditConsumerConfig, z *ZK) (host string, port string, err error) {

	if conf.AuditSvcIP != "" && conf.AuditSvcPort != "" {
		return conf.AuditSvcIP, conf.AuditSvcPort, nil
	}

	// private type models the data we will see in ZK
	type addr struct {
		Host string
		Port int
	}
	type endpoint struct {
		Endpoint addr `json:"serviceEndpoint"`
	}

	children, _, err := z.Conn().Children(conf.AuditZKPath)
	if err != nil {
		return "", "", err
	}
	var auditors []endpoint
	for _, c := range children {
		data, _, err := z.Conn().Get(conf.AuditZKPath + "/" + c)
		if err != nil {
			break
		}
		var e endpoint
		if err := json.Unmarshal(data, &e); err != nil {
			break
		}
		auditors = append(auditors, e)
	}
	if len(auditors) < 1 {
		return "", "", fmt.Errorf("could not find audit-transformation service at path: %s", conf.AuditZKPath)
	}

	// Pick a random endpoint
	i := randomIndex(len(auditors))
	discovered := auditors[i].Endpoint
	ipAddr, err := net.ResolveIPAddr("ip4", discovered.Host)
	if err != nil {
		return "", "", err
	}
	return ipAddr.IP.String(), strconv.Itoa(discovered.Port), nil
}

func randomIndex(length int) int {
	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s)
	return r.Intn(length)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Audit wraps an audit service connection.
type Audit struct {
	conn      *auditsvc.AuditTransformationServiceClient
	m         *sync.RWMutex
	reconnect bool
}

// Conn returns the inner audit service connection.
func (a *Audit) Conn() *auditsvc.AuditTransformationServiceClient {
	a.m.RLock()
	defer a.m.RUnlock()
	return a.conn
}

// SetConn sets the inner audit service connection.
func (a *Audit) SetConn(conn *auditsvc.AuditTransformationServiceClient) {
	a.m.Lock()
	a.conn = conn
	a.m.Unlock()
}

// ZK wraps a zookeeper connection.
type ZK struct {
	conn *zk.Conn
	m    *sync.RWMutex
}

// Conn gives us a connection, and prevents reads during writes.
func (z *ZK) Conn() *zk.Conn {
	z.m.RLock()
	defer z.m.RUnlock()
	return z.conn
}

// SetConn sets the inner Zookeeper connection.
func (z *ZK) SetConn(conn *zk.Conn) {
	z.m.Lock()
	z.conn = conn
	z.m.Unlock()
}

// Kafka wraps a kafka consumer.
type Kafka struct {
	c sarama.Consumer
	m *sync.RWMutex
}

// Consumer returns the inner consumer, and prevents reads during writes.
func (k *Kafka) Consumer() sarama.Consumer {
	k.m.RLock()
	defer k.m.RUnlock()
	return k.c
}

// SetConsumer sets the inner consumer.
func (k *Kafka) SetConsumer(c sarama.Consumer) {
	k.m.Lock()
	k.c = c
	k.m.Unlock()
}
