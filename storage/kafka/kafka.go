package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	kafka "github.com/Shopify/sarama"
	"github.com/wenlian/remote-tail/command"
	"github.com/wenlian/remote-tail/storage"
)

func init() {
	storage.RegisterStorageDriver("kafka", new)
}

type kafkaStorage struct {
	producer    kafka.AsyncProducer
	topic       string
	machineName string
}

//chan command.Message
func (driver *kafkaStorage) infoToDetailSpec(msg command.Message) *command.Message {
	message := &command.Message{
		Host:    msg.Host,
		Path:    msg.Path,
		Content: strings.Replace(msg.Content, "\r\n", "", -1),
	}
	return message

}
func (driver *kafkaStorage) AddStats(msg command.Message, brokers string) error {
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
	var cfg command.Config
	if _, err := toml.DecodeFile("config.toml", &cfg); err != nil {
		log.Fatal(err)
	}

	if err != nil {
		return nil, err
	}
	return newStorage(machineName, cfg)
}

func generateTLSConfig(cfg command.Config) (*tls.Config, error) {
	if cfg.KafkaCertfile != "" && cfg.KafkaKeyfile != "" && cfg.KafkaCafile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.KafkaCertfile, cfg.KafkaKeyfile)
		if err != nil {
			return nil, err
		}

		caCert, err := ioutil.ReadFile(cfg.KafkaCafile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		return &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: cfg.KafkaVerifySSL,
		}, nil
	}

	return nil, nil
}

func newStorage(machineName string, cfg command.Config) (storage.StorageDriver, error) {
	config := kafka.NewConfig()

	tlsConfig, err := generateTLSConfig(cfg)
	if err != nil {
		return nil, err
	}

	if tlsConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}

	config.Producer.RequiredAcks = kafka.WaitForAll
	config.Producer.Flush.Frequency = 3 * time.Second // Flush batches every 3s

	brokerList := strings.Split(cfg.KafkaBrokers, ",")

	producer, err := kafka.NewAsyncProducer(brokerList, config)
	if err != nil {
		return nil, err
	}
	ret := &kafkaStorage{
		producer:    producer,
		topic:       cfg.KafkaTopic,
		machineName: machineName,
	}
	return ret, nil
}
