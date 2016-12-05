package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"strings"
	"time"

	kafka "github.com/Shopify/sarama"
	"github.com/wenlian/remote-tail/command"
	"github.com/wenlian/remote-tail/storage"
)

func init() {
	storage.RegisterStorageDriver("kafka", new)
}

var (
	brokers   = flag.String("storage_driver_kafka_broker_list", "docker52:9092", "kafka broker(s) csv")
	topic     = flag.String("storage_driver_kafka_topic", "remote-tail", "kafka topic")
	certFile  = flag.String("storage_driver_kafka_ssl_cert", "", "optional certificate file for TLS client authentication")
	keyFile   = flag.String("storage_driver_kafka_ssl_key", "", "optional key file for TLS client authentication")
	caFile    = flag.String("storage_driver_kafka_ssl_ca", "", "optional certificate authority file for TLS client authentication")
	verifySSL = flag.Bool("storage_driver_kafka_ssl_verify", true, "verify ssl certificate chain")
)

type kafkaStorage struct {
	producer    kafka.AsyncProducer
	topic       string
	machineName string
}

//chan command.Message
func (driver *kafkaStorage) infoToDetailSpec(msg command.Message) *command.Message {
	//timestamp := time.Now()

	message := &command.Message{
		Host:    msg.Host,
		Path:    msg.Path,
		Content: msg.Content,
	}
	return message

}
func (driver *kafkaStorage) AddStats(msg command.Message) error {
	detail := driver.infoToDetailSpec(msg)
	b, err := json.Marshal(detail)
	driver.producer.Input() <- &kafka.ProducerMessage{
		Topic: driver.topic,
		Value: kafka.StringEncoder(b),
	}
	return err

}

func (self *kafkaStorage) Close() error {
	return self.producer.Close()
}

func new() (storage.StorageDriver, error) {
	machineName, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(machineName)
}

func generateTLSConfig() (*tls.Config, error) {
	if *certFile != "" && *keyFile != "" && *caFile != "" {
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			return nil, err
		}

		caCert, err := ioutil.ReadFile(*caFile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		return &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: *verifySSL,
		}, nil
	}

	return nil, nil
}

func newStorage(machineName string) (storage.StorageDriver, error) {
	config := kafka.NewConfig()

	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}

	config.Producer.RequiredAcks = kafka.WaitForAll
	config.Producer.Flush.Frequency = 3 * time.Second // Flush batches every 3s

	brokerList := strings.Split(*brokers, ",")

	producer, err := kafka.NewAsyncProducer(brokerList, config)
	if err != nil {
		return nil, err
	}
	ret := &kafkaStorage{
		producer:    producer,
		topic:       *topic,
		machineName: machineName,
	}
	return ret, nil
}
