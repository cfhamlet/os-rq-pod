package config

// EnvPrefix for env
const EnvPrefix = "RQ_POD"

// DefaultConfig TODO
var DefaultConfig = map[string]interface{}{
	"debug":               false,
	"log.level":           "info",
	"http.addr":           ":6789",
	"http.log.enable":     true,
	"http.api.path":       "",
	"redis.addr":          "localhost:6379",
	"redis.password":      "",
	"redis.db":            0,
	"redis.max_mem_limit": 256 << 20,
}
