package config

import (
	"fmt"
	"os"
	"strings"

	"encoding/json"

	"github.com/spf13/viper"
)

// LoadDefaultConfig TODO
func LoadDefaultConfig(conf *viper.Viper, defaultConfig map[string]interface{}) {
	for k, v := range defaultConfig {
		conf.SetDefault(k, v)
	}
}

// LoadConfig TODO
func LoadConfig(conf *viper.Viper, envPrefix string, defaultConfig map[string]interface{}) error {
	LoadDefaultConfig(conf, defaultConfig)
	if conf.IsSet("configfile") {
		conf.SetConfigFile(conf.GetString("configfile"))
	}
	conf.SetEnvPrefix(envPrefix)
	conf.AutomaticEnv()
	conf.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := conf.ReadInConfig(); err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

// PrintDebugConfig TODO
func PrintDebugConfig(conf *viper.Viper) {
	debug := false
	if conf.IsSet("debug") {
		debug = conf.GetBool("debug")
	}
	if debug {
		o, e := json.MarshalIndent(conf.AllSettings(), " ", "    ")
		if e != nil {
			panic(fmt.Errorf("%v", e))

		}
		fmt.Fprintf(os.Stderr, "Config:\n%s\n", o)
	}
}

// Configurable TODO
type Configurable interface {
	Conf() *viper.Viper
}
