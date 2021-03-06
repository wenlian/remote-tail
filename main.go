package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	//	"github.com/samuel/go-zookeeper/zk"
	"github.com/wenlian/remote-tail/command"
	"github.com/wenlian/remote-tail/console"
	"github.com/wenlian/remote-tail/storage"
	_ "github.com/wenlian/remote-tail/storage/kafka"
)

var mossSep = ".--. --- .-- . .-. . -..   -... -.--   -- -.-- .-.. -..- ... .-- \n"

var welcomeMessage string = getWelcomeMessage() + console.ColorfulText(console.TextMagenta, mossSep)

var filePath *string = flag.String("file", "", "eg:-file=\"/home/data/logs/**/*.log\"")
var hostStr *string = flag.String("hosts", "", "eg:-hosts=root@192.168.1.225,root@192.168.1.226")
var configFile *string = flag.String("conf", "config.toml", "-conf=config.toml")
var zkServer *string = flag.String("zk", "", "-zk 192.168.76.52:2181")

func usageAndExit(message string) {

	if message != "" {
		fmt.Fprintln(os.Stderr, message)
	}

	flag.Usage()
	fmt.Fprint(os.Stderr, "\n")

	os.Exit(1)
}

func printWelcomeMessage(config command.Config) {
	fmt.Println(welcomeMessage)
	for _, server := range config.Servers {
		// If there is no tail_file for a service configuration, the global configuration is used
		if server.TailFile == "" {
			server.TailFile = config.TailFile
		}
		serverInfo := fmt.Sprintf("%s@%s:%s", server.User, server.Hostname, server.TailFile)
		fmt.Println(console.ColorfulText(console.TextMagenta, serverInfo))
	}
	fmt.Printf("\n%s\n", console.ColorfulText(console.TextCyan, mossSep))
}

func parseConfig(filePath string, hostStr string, configFile string) (config command.Config) {
	//本地配置文件不为空，读取本地文件
	if configFile != "" {
		if _, err := toml.DecodeFile(configFile, &config); err != nil {
			log.Fatal(err)
		}

	} else {
		//本地文件为空，读取zk
		var hosts []string = strings.Split(hostStr, ",")

		config = command.Config{}
		config.TailFile = filePath
		config.Servers = make(map[string]command.Server, len(hosts))
		for index, hostname := range hosts {
			hostInfo := strings.Split(strings.Replace(hostname, ":", "@", -1), "@")
			var port int
			if len(hostInfo) > 2 {
				port, _ = strconv.Atoi(hostInfo[2])
			}
			config.Servers["server_"+string(index)] = command.Server{
				ServerName: "server_" + string(index),
				Hostname:   hostInfo[1],
				User:       hostInfo[0],
				Port:       port,
			}
		}
	}

	return
}

func main() {

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, welcomeMessage)
		fmt.Fprint(os.Stderr, "Options:\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if (*filePath == "" || *hostStr == "") && *configFile == "" {
		usageAndExit("")
	}

	config := parseConfig(*filePath, *hostStr, *configFile)

	printWelcomeMessage(config)

	outputs := make(chan command.Message, 255)
	var wg sync.WaitGroup

	for _, server := range config.Servers {
		wg.Add(1)
		go func(server command.Server) {
			defer func() {
				if err := recover(); err != nil {
					fmt.Printf(console.ColorfulText(console.TextRed, "Error: %s\n"), err)
				}
			}()
			defer wg.Done()

			// If there is no tail_file for a service configuration, the global configuration is used
			if server.TailFile == "" {
				server.TailFile = config.TailFile
			}

			// If the service configuration does not have a port, the default value of 22 is used
			if server.Port == 0 {
				server.Port = 22
			}
			pathArr := strings.Split(server.TailFile, ";")
			log.Println(pathArr)
			for _, logPath := range pathArr {
				if len(logPath) > 2 {
					/******************************************
					* 日志等级判断
					* 0. info 关键词:INFO
					* 1. warn 关键词:WARN
					* 2. err 关键词:ERR
					* default. 所有log
					******************************************/
					cmd := command.NewCommand(server, logPath, config.LogLevel)
					wg.Add(1)
					go func() {
						cmd.Execute(outputs)
					}()
				}
			}
		}(server)
	}

	if len(config.Servers) > 0 {
		//
		if config.StorageDriver != "" {
			backendStorage, err := storage.New(config.StorageDriver)
			if err != nil {
				log.Println(err)
			}
			go func() {
				for output := range outputs {
					fmt.Printf(
						"%s %s %s %s",
						console.ColorfulText(console.TextGreen, output.Host),
						console.ColorfulText(console.TextGreen, output.Path),
						console.ColorfulText(console.TextYellow, "->"),
						output.Content,
					)
					if output.Content != "" && output.Content != "\r\n" && config.StorageDriver != "" {
						backendStorage.AddStats(output, config.KafkaBrokers)
					}
				}
			}()
		} else {
			log.Println("Storage Type nil!")
		}
	} else {
		fmt.Println(console.ColorfulText(console.TextRed, "No target host is available"))
	}

	wg.Wait()
}
