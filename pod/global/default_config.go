package global

// EnvPrefix for env
const EnvPrefix = "RQ_POD"

// DefaultConfig TODO
var DefaultConfig = map[string]interface{}{
	"debug":              false,
	"log.level":          "info",
	"http.addr":          ":6789",
	"http.log.enable":    true,
	"http.api.path":      "",
	"redis.addr":         "localhost:6379",
	"redis.password":     "",
	"redis.db":           0,
	"limit.http.size":    1 << 20,
	"limit.redis.memory": int64(1 << 30),
	"limit.request.num":  10000000,
	"limit.queue.num":    1000,
	"limit.queue.length": 10000,
}
