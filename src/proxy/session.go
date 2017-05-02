package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
	log "github.com/cihub/seelog"
	"github.com/carlvine500/redis-go-cluster"
	"util"
)

type ClientSession interface {
	ParseRequest() ([]interface{}, error)
	Response(reply interface{}, err error, cmd string)
	RemoteAddr() string
	Close() error
}

func NewSession(netConn net.Conn, readTimeout, writeTimeout int64) ClientSession {
	//atomic.

	return &clientSession{
		conn:         netConn,
		bufferWriter: bufio.NewWriter(netConn),
		bufferReader: bufio.NewReader(netConn),
		readTimeout:  time.Duration(readTimeout) * time.Millisecond,
		writeTimeout: time.Duration(readTimeout) * time.Millisecond,
	}
}

type clientSession struct {
	conn         net.Conn
	readTimeout  time.Duration
	bufferReader *bufio.Reader
	writeTimeout time.Duration
	bufferWriter *bufio.Writer
}

func (clientConn *clientSession) RemoteAddr() string {
	return clientConn.conn.RemoteAddr().String()
}

func (clientConn *clientSession) Close() error {
	return clientConn.conn.Close()
}

func (clientConn *clientSession) ParseRequest() ([]interface{}, error) {
	//bad performance
	/*if clientConn.readTimeout > 0 {
		clientConn.conn.SetReadDeadline(time.Now().Add(clientConn.readTimeout))
	}*/
	line, err := readLine(clientConn.bufferReader)
	if err != nil {
		return nil, err
	}

	if line[0] != '*' {
		return nil, ProtocolError(fmt.Sprintf("bad command %s ", string(line)))
	}
	lineCount, err := strconv.ParseInt(string(line[1:]), 10, 64)
	if err != nil {
		return nil, err
	}

	results := make([]interface{}, lineCount)
	for i := range results {
		line, err = readLine(clientConn.bufferReader)
		if line[0] != '$' {
			return nil, ProtocolError(fmt.Sprintf("redis: expected '$', but got %q", line))
		}
		argLen, err := parseInt(line[1:])
		if err != nil {
			return nil, err
		}
		p, err := readLen(clientConn.bufferReader, argLen)
		if err != nil {
			return nil, err
		}
		if line, err := readLine(clientConn.bufferReader); err != nil {
			return nil, err
		} else if len(line) != 0 {
			return nil, ProtocolError("bad bulk string format")
		}
		results[i] = p
	}
	return results, nil
}

func readLine(br *bufio.Reader) ([]byte, error) {
	p, err := br.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		return nil, ProtocolError("response longer that buffer")
	}
	if err != nil {
		// such as net.OpError,conn close/reset
		return nil, err
	}
	i := len(p) - 2
	if i < 0 || p[i] != '\r' {
		return nil, ProtocolError("bad response length or line terminator")
	}
	return p[:i], nil
}

func readLen(br *bufio.Reader, len int64) ([]byte, error) {
	p := make([]byte, len)
	_, err := io.ReadFull(br, p)
	return p, err
}

func parseInt(p []byte) (int64, error) {
	if len(p) == 0 {
		return 0, ProtocolError("malformed integer")
	}
	var negate bool
	if p[0] == '-' {
		negate = true
		p = p[1:]
		if len(p) == 0 {
			return 0, ProtocolError("malformed integer")
		}
	}
	var n int64
	for _, b := range p {
		n *= 10
		if b < '0' || b > '9' {
			return 0, ProtocolError("illegal bytes in length")
		}
		n += int64(b - '0')
	}
	if negate {
		n = -n
	}
	return n, nil
}

func (clientConn *clientSession)Response(reply interface{}, err error, cmd string) {
	ParseReply(clientConn.bufferWriter, reply, err, cmd)
	// bad performance
	/*if clientConn.writeTimeout > 0 {
		clientConn.conn.SetWriteDeadline(time.Now().Add(clientConn.writeTimeout))
	}*/
	if flushErr := clientConn.bufferWriter.Flush(); flushErr != nil {
		log.Error(flushErr)
	}
}

// TODO multi-exe pipeline 指令的支持
func ParseReply(bw *bufio.Writer, reply interface{}, err error, cmd string) {
	if err != nil {
		bw.WriteString(fmt.Sprintf("-%s\r\n", err.Error()))
		return
	}
	if reply == nil {
		bw.Write([]byte("$-1\r\n"))
		return
	} else if redisErr, ok := reply.(redis.RedisError); ok {
		bw.WriteString(fmt.Sprintf("-%s\r\n", redisErr.Error()))
		return
	}

	switch cmd {
	case "HMSET", //
		"LSET", //
		"PING", //
		"LTRIM", //
		"MSET", //
		"PFMERGE", //
		"PSETEX", //
		"RENAME", //
		"RESTORE", //
		"SET", //
		"SETEX", //
		"TYPE":    //
		getStatusCodeReply(bw, reply)
	case "HINCRBYFLOAT", //
		"INCRBYFLOAT", //
		"ZINCRBY", //
		"ZSCORE", //
		"BRPOPLPUSH", //
		"ECHO", //
		"GET", //
		"GETRANGE", //
		"GETSET", //
		"HGET", //
		"LINDEX", //
		"LPOP", //
		"OBJECTENCODING", //
		"RPOP", //
		"RPOPLPUSH", //
		"DUMP", //
		"SUBSTR":         //
		getBinaryBulkReply(bw, reply)
	case "BLPOP", //
		"BRPOP", //
		"HGETALL", //
		"HKEYS", //
		"HMGET", //
		"HVALS", //
		"KEYS", //
		"LRANGE", //
		"MGET", //
		"SDIFF", //
		"SINTER", //
		"SMEMBERS", //
		"SUNION", //
		"ZRANGE", //
		"ZRANGEBYLEX", //
		"ZRANGEBYSCORE", //
		"ZREVRANGE", //
		"ZREVRANGEBYLEX", //
		"ZREVRANGEBYSCORE":
		getBinaryMultiBulkReply(bw, reply)
	case "APPEND", //
		"BITCOUNT", //
		"BITOP", //
		"BITPOS", //
		"CLUSTERCOUNTKEYSINSLOT", //
		"CLUSTERKEYSLOT", //
		"DECR", //
		"DECRBY", //
		"DEL", //
		"EXISTS", //
		"EXPIRE", //
		"EXPIREAT", //
		"GETBIT", //
		"HDEL", //
		"HEXISTS", //
		"HINCRBY", //
		"HLEN", //
		"HSET", //
		"HSETNX", //
		"INCR", //
		"INCRBY", //
		"LINSERT", //
		"LLEN", //
		"LPUSH", //
		"LPUSHX", //
		"LREM", //
		"MOVE", //
		"MSETNX", //
		"OBJECTIDLETIME", //
		"OBJECTREFCOUNT", //
		"PERSIST", //
		"PEXPIRE", //
		"PEXPIREAT", //
		"PFADD", //
		"PFCOUNT", //
		"PTTL", //
		"PUBLISH", //
		"PUBSUBNUMPAT", //
		"RENAMENX", //
		"RPUSH", //
		"RPUSHX", //
		"SADD", //
		"SCARD", //
		"SDIFFSTORE", //
		"SENTINELRESET", //
		"SETBIT", //
		"SETNX", //
		"SETRANGE", //
		"SINTERSTORE", //
		"SISMEMBER", //
		"SMOVE", //
		"SREM", //
		"STRLEN", //
		"SUNIONSTORE", //
		"TTL", //
		"WAITREPLICAS", //
		"ZADD", //
		"ZCARD", //
		"ZCOUNT", //
		"ZINTERSTORE", //
		"ZLEXCOUNT", //
		"ZRANK", //
		"ZREM", //
		"ZREMRANGEBYLEX", //
		"ZREMRANGEBYRANK", //
		"ZREMRANGEBYSCORE", //
		"ZREVRANK", //
		"ZUNIONSTORE":
		getIntegerReply(bw, reply)
	case "HSCAN", //
		"SSCAN", //
		"ZSCAN": //
		getScanReply(bw, reply)
	case "SPOP", "SRANDMEMBER":
		if _, ok := reply.([]byte); ok {
			getBinaryBulkReply(bw, reply)
		} else {
			getBinaryMultiBulkReply(bw, reply)
		}
	case "SORT":
		if _, ok := reply.(int64); ok {
			getIntegerReply(bw, reply)
		} else {
			getBinaryMultiBulkReply(bw, reply)
		}
	default:
		bw.WriteString(fmt.Sprintf("-%s\r\n", "unSupported command=" + cmd))
	}
}

func getStatusCodeReply(bw *bufio.Writer, reply interface{}) {
	stringReply := reply.(string)
	bw.WriteByte('+')
	bw.WriteString(stringReply)
	bw.WriteString("\r\n")
}

func getErrorReply(bw *bufio.Writer, errStr string) {
	bw.WriteByte('-')
	bw.WriteString(errStr)
	bw.WriteString("\r\n")
}

func getIntegerReply(bw *bufio.Writer, reply interface{}) {
	intReply := reply.(int64)
	bw.WriteByte(':')
	bw.WriteString(strconv.FormatInt(intReply, 10))
	bw.WriteString("\r\n")
}

func getBinaryBulkReply(bw *bufio.Writer, reply interface{}) {
	byteReply := reply.([]byte)
	bw.WriteByte('$')
	bw.WriteString(util.Itoa(len(byteReply)))
	bw.WriteString("\r\n")
	bw.Write(byteReply)
	bw.WriteString("\r\n")
}
func getBinaryMultiBulkReply(bw *bufio.Writer, reply interface{}) {
	arrayReply := reply.([]interface{})
	bw.WriteByte('*')
	bw.WriteString(util.Itoa(len(arrayReply)))
	bw.WriteString("\r\n")
	for _, v := range arrayReply {
		getBinaryBulkReply(bw, v)
	}
}

func getScanReply(bw *bufio.Writer, reply interface{}) {
	bw.WriteByte('*')
	bw.WriteByte('2')
	bw.WriteString("\r\n")
	arrayReply := reply.([]interface{})
	getBinaryBulkReply(bw, arrayReply[0])
	getBinaryMultiBulkReply(bw, arrayReply[1])
}
