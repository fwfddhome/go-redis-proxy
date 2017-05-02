package proxy

/**map[command]forbidden*/
var commands = map[string]bool{
	"CLUSTER":   true, //
	"READONLY":  true, //
	"READWRITE": true, //

	"AUTH":   true, //
	"ECHO":   true, //
	"SELECT": true, //

	"PFADD":   true, //
	"PFCOUNT": true, //
	"PFMERGE": true, //

	"KEYS":      true, //
	"MIGRATE":   true, //
	"MOVE":      true, //
	"OBJECT":    true, //
	"RANDOMKEY": true, //
	"RENAME":    true, //
	"RENAMENX":  true, //
	"WAIT":      true, //
	"SCAN":      true, //

	"BLPOP":      true, //
	"BRPOP":      true, //
	"BRPOPLPUSH": true, //
	"RPOPLPUSH":  true, //

	"PSUBSCRIBE":   true, //
	"PUBSUB":       true, //
	"PUBLISH":      true, //
	"PUNSUBSCRIBE": true, //
	"SUBSCRIBE":    true, //
	"UNSUBSCRIBE":  true, //

	"EVAL":    true, //
	"EVALSHA": true, //
	"SCRIPT":  true, //

	"BGREWRITEAOF": true, //
	"BGSAVE":       true, //
	"CLIENT":       true, //
	"COMMAND":      true, //
	"CONFIG":       true, //
	"DBSIZE":       true, //
	"DEBUG":        true, //

	"FLUSHALL": true, //
	"FLUSHDB":  true, //
	"INFO":     true, //
	"LASTSAVE": true, //
	"MONITOR":  true, //
	"ROLE":     true, //
	"SAVE":     true, //
	"SHUTDOWN": true, //
	"SLAVEOF":  true, //
	"SLOWLOG":  true, //
	"SYNC":     true, //
	"TIME":     true, //

	"SDIFF":       true, //
	"SDIFFSTORE":  true, //
	"SINTER":      true, //
	"SINTERSTORE": true, //
	"SMOVE":       true, //
	"SUNION":      true, //
	"SUNIONSTORE": true, //

	"ZINTERSTORE": true, //
	"ZUNIONSTORE": true, //
	"BITOP":       true, //

}

func IsCmdForbidden(cmd string) bool {
	supported, ok := commands[cmd]
	if ok {
		return supported
	} else {
		return false
	}
}
