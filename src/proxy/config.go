package proxy

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	log "github.com/cihub/seelog"
	"encoding/json"
	"strings"
	"net"
)

//attention! .yaml just support lowercase.
type ProxyConfig struct {
	Zookeeper_servers    []string
	Cpu_num              int
	Client_connections   int
	Timeout              int
	Backlog              int
	Server_retry_timeout int
	Server_failure_limit int
	Proxy_clusters       []ProxyClusterConfig
}
type ProxyClusterConfig struct {
	Cluster              string
	Listen               string
	Servers              []string
	Prefix               string

	Client_connections   int
	Timeout              int
	Backlog              int
	Server_retry_timeout int
	Server_failure_limit int
	PrefixBytes          []byte
}

func NewProxyConfig() *ProxyConfig {
	config := &ProxyConfig{
		Cpu_num:0,
		Client_connections:102400,
		Timeout:2000,
		Backlog:1024,
		Server_retry_timeout:200,
		Server_failure_limit:2,
	}
	filepath, _ := filepath.Abs("./redis.yaml")
	log.Infof("config filepath: %s", filepath)
	file, err := os.Open(filepath)
	if err != nil {
		log.Errorf("error: %v", err)
		os.Exit(1)
	}
	data, err := ioutil.ReadAll(file)
	log.Infof("config yaml:\n---------- ----------\n%s\n---------- ----------", string(data))
	if err != nil {
		log.Errorf("error: %v", err)
		os.Exit(1)
	}
	err = yaml.Unmarshal([]byte(data), config)
	for i, _ := range config.Proxy_clusters {
		pc := &config.Proxy_clusters[i]
		if pc.Client_connections <= 0 {
			pc.Client_connections = config.Client_connections
		}
		if pc.Timeout <= 0 {
			pc.Timeout = config.Timeout
		}
		if pc.Backlog <= 0 {
			pc.Backlog = config.Backlog
		}
		if pc.Server_retry_timeout <= 0 {
			pc.Server_retry_timeout = config.Server_retry_timeout
		}
		if pc.Server_failure_limit <= 0 {
			pc.Server_failure_limit = config.Server_failure_limit
		}
		if (pc.Prefix != "") {
			pc.PrefixBytes = []byte(pc.Prefix)
		}
		pc.Listen = strings.Replace(pc.Listen, "localhost", getOutboundIP(), 1)
		pc.Listen = strings.Replace(pc.Listen, "127.0.0.1", getOutboundIP(), 1)
	}
	jsonResult, _ := json.Marshal(config)
	log.Infof("config json:\n%v", string(jsonResult))
	if err != nil {
		log.Errorf("error: %v", err)
		os.Exit(1)
	}
	return config
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Error(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")

	return localAddr[0:idx]
}
