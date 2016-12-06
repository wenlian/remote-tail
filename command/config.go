package command

type Server struct {
	ServerName string `toml:"server_name"`
	Hostname   string `toml:"hostname"`
	Port       int    `toml:"port"`
	User       string `toml:"user"`
	Password   string `toml:"password"`
	TailFile   string `toml:"tail_file"`
}

type Config struct {
	TailFile       string            `toml:"tail_file"`
	LogLevel       int               `toml:"log_level"`
	StorageDriver  string            `toml:"storage_driver"`
	KafkaBrokers   string            `toml:"kafka_brokers"`
	KafkaTopic     string            `toml:"kafka_topic"`
	KafkaCertfile  string            `toml:"kafka_certfile"`
	KafkaKeyfile   string            `toml:"kafka_keyfile"`
	KafkaCafile    string            `toml:"kafka_cafile"`
	KafkaVerifySSL bool              `toml:"kafka_verifyssl"`
	Servers        map[string]Server `toml:"servers"`
}
