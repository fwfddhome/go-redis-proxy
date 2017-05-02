package main

import (
	// _ "net/http/pprof"
	"net/http"
	"proxy"
	"runtime"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/samuel/go-zookeeper/zk"
)

func main() {
	//initMonitor(config)
	config := proxy.NewProxyConfig()
	initLog()
	initCpu(config)

	zkConn := connectZookeeper(config.Zookeeper_servers)

	log.Info("connect Zookeeper end!")
	for i := 0; i < len(config.Proxy_clusters); i++ {
		go proxy.StartRedisClusterProxy(&config.Proxy_clusters[i], zkConn)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

func connectZookeeper(zks []string) *zk.Conn {
	log.Info("connect Zookeeper started!")
	conn, _, err := zk.Connect(zks, 60*time.Second)
	proxy.Must(err)
	return conn
}
func initCpu(config *proxy.ProxyConfig) {
	log.Info("initCpu started!")
	if config.Cpu_num > 0 {
		runtime.GOMAXPROCS(config.Cpu_num)
	} else {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
}

func initMonitor(config *proxy.ProxyConfig) {
	log.Info("monitor started!")
	go func() {
		log.Info(http.ListenAndServe("localhost:6060", nil))
	}()
}

func initLog() {
	logger, err := log.LoggerFromConfigAsFile("./seelog.xml")
	if err != nil {
		log.Critical("err parsing config log file", err)
		return
	}
	log.ReplaceLogger(logger)
	log.Info("seelog started!")
}
