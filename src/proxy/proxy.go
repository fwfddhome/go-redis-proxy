package proxy

import (
	"github.com/carlvine500/redis-go-cluster"
	"github.com/samuel/go-zookeeper/zk"
	log "github.com/cihub/seelog"
	"net"
	"strings"
	"time"
	"os"
	"os/signal"
	"syscall"
	"bytes"
	"fmt"
)

func StartRedisClusterProxy(proxyCluster *ProxyClusterConfig, zkConn *zk.Conn) {

	ln, err := net.Listen("tcp4", proxyCluster.Listen)
	if err != nil {
		log.Error(err.Error())
	}
	redisCluster, _ := createRedisCluster(proxyCluster)
	registerIntoZookeeper(zkConn, proxyCluster)
	signalNotify(zkConn, redisCluster)

	channels := make(chan net.Conn, proxyCluster.Client_connections)
	go func() {
		for conn := range channels {
			go handleRequest(conn, redisCluster, proxyCluster)
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error("accept error", err.Error())
		} else {
			channels <- conn
		}
	}

}

func signalNotify(zkConn *zk.Conn, redisCluster *redis.Cluster) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, os.Kill)
	go func() {
		for sig := range c {
			log.Infof("Got A %v Signal! ", sig.String())
			zkConn.Close()
			redisCluster.Close()
			GMonitor.Close()
			log.Flush()
			os.Exit(1)
		}
	}()
}

func registerIntoZookeeper(zkConn *zk.Conn, proxyCluster *ProxyClusterConfig) {
	if zkConn == nil {
		return
	}
	proxyClusterPath := "/gcache/proxy" + "/" + proxyCluster.Cluster
	ensureExsists(zkConn, proxyClusterPath)

	serverPath := proxyClusterPath + "/" + proxyCluster.Listen
	_, err := zkConn.Create(serverPath, []byte("here"), int32(zk.FlagEphemeral), zk.WorldACL(zk.PermAll))
	if err != nil && strings.Contains(err.Error(), "node already exists") {
		//skip
	} else {
		Must(err)
	}
	log.Infof("registed into zookeeper,path=%s", serverPath)
}

func ensureExsists(zkConn *zk.Conn, path string) {
	pathArr := strings.Split(path, "/")
	existsPathDepth := 0
	for i := len(pathArr); i > 0; i-- {
		tmpPath := strings.Join(pathArr[0:i], "/")
		exist, _, err := zkConn.Exists(tmpPath)
		Must(err)
		if (exist) {
			existsPathDepth = i
			break
		}
	}
	for i := existsPathDepth + 1; i < len(pathArr) + 1; i++ {
		tmpPath := strings.Join(pathArr[0:i], "/")
		_, err := zkConn.Create(tmpPath, []byte{}, int32(0), zk.WorldACL(zk.PermAll))
		Must(err)
	}
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func handleRequest(conn net.Conn, redisCluster *redis.Cluster, proxyCluster *ProxyClusterConfig) {
	session := NewSession(conn, -1, -1)
	log.Infof("connection created, remote:%v, at:%v", session.RemoteAddr(), time.Now().Format(time.Stamp))
	for {
		beginTime := time.Now().UnixNano()
		request, reqErr := session.ParseRequest()

		if reqErr != nil {
			if protocolErr, ok := reqErr.(ProtocolError); ok {
				session.Response(nil, protocolErr, "")
				return
			}
			session.Close()
			log.Infof("connection closed, remote:%v, at:%v", session.RemoteAddr(), time.Now().Format(time.Stamp))
			return
		}

		var cmd = strings.ToUpper(strings.TrimSpace(string(request[0].([]uint8))))
		defer func() {
			if err := recover(); err != nil {
				session.Response(nil, ProtocolError(fmt.Sprintf("Unknow Error,%v\r\n")), cmd)
				log.Errorf("Unknow Error[E],cmd=%s,request=%v,error=%s", cmd, request, err)
			}
		}()

		var args []interface{} = request[1:]
		var reply, err = process(redisCluster, cmd, proxyCluster.PrefixBytes, args...)

		endTime := time.Now().UnixNano()
		GMonitor.RecordOneAndRt(proxyCluster.Listen, int((endTime - beginTime) / 1000 / 1000))
		session.Response(reply, err, cmd)
	}
}

func B2S(bs []uint8) string {
	ba := []byte{}
	for _, b := range bs {
		ba = append(ba, byte(b))
	}
	return string(ba)
}

func process(redisCluster *redis.Cluster, cmd string, prefixBytes []byte, args ...interface{}) (interface{}, error) {
	// TODO avoid string 处理逻辑,全部使用bytes
	switch {
	case IsCmdForbidden(cmd):
		return nil, ProtocolError("unsupported cmd " + cmd)
	case cmd == "QUIT":
		return nil, ProtocolError("client issue QUIT")
	case cmd == "PING":
		return "PONG", nil
	}

	if (prefixBytes != nil) {
		args[0] = bytes.Join([][]byte{prefixBytes, args[0].([]byte)}, []byte(":"))
	}
	// TODO 慢查询，性能统计页面
	return redisCluster.Do(cmd, args...)
}

func createRedisCluster(proxyCluster *ProxyClusterConfig) (*redis.Cluster, error) {
	cluster, err := redis.NewCluster(
		&redis.Options{
			StartNodes:   proxyCluster.Servers,
			ConnTimeout:  time.Duration(proxyCluster.Timeout) * time.Millisecond,
			ReadTimeout:  time.Duration(proxyCluster.Server_retry_timeout) * time.Millisecond,
			WriteTimeout: time.Duration(proxyCluster.Server_retry_timeout) * time.Millisecond,
			KeepAlive:    proxyCluster.Backlog,
			AliveTime:    60 * time.Second,
		})

	return cluster, err
}


