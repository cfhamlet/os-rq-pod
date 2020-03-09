package log

import (
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var rootLoggerName string = "RQ"

// Logger TODO
var Logger *logging.Logger = logging.MustGetLogger(rootLoggerName)

type configLoggingFunc func(*viper.Viper) error

// Default TODO
const (
	DefaultLoggerName    = "RQ"
	DefaultLogTimeFormat = "2006-01-02 15:04:05"
	DefaultLogLevel      = "debug"
	DefaultLogOut        = "stderr"
	DefaultLogFormat     = `[%{time:` + DefaultLogTimeFormat + `}] [%{module}] [%{level}] %{message}`
)

func init() {
	var out io.Writer

	switch DefaultLogOut {
	case "stderr":
		out = os.Stderr
	case "stdout":
		out = os.Stdout
	default:
		out = ioutil.Discard
	}

	backend := logging.NewLogBackend(out, "", 0)
	logging.SetBackend(backend)

	l := logging.GetLevel(DefaultLogLevel)
	logging.SetLevel(l, DefaultLoggerName)

	formatter := logging.MustStringFormatter(DefaultLogFormat)
	logging.SetFormatter(formatter)
}

// ConfigLogging TODO
func ConfigLogging(conf *viper.Viper) error {
	for _, f := range []configLoggingFunc{
		ConfigLoggingBackend,
		ConfigLoggingFormat,
		ConfigLoggingLevel,
	} {
		if err := f(conf); err != nil {
			return err
		}
	}
	return nil
}

// ConfigLoggingLevel TODO
func ConfigLoggingLevel(conf *viper.Viper) error {
	level := "info"
	if conf.IsSet("log.level") {
		level = conf.GetString("log.level")
	}
	debug := false
	if conf.IsSet("debug") {
		debug = conf.GetBool("debug")
	}

	return SetLoggingLevel(level, debug)
}

// ConfigLoggingFormat TODO
func ConfigLoggingFormat(conf *viper.Viper) error {
	if !conf.IsSet("log.format") {
		return nil
	}
	format := conf.GetString("log.format")
	return SetLoggingFormat(format)
}

// ConfigLoggingBackend TODO
func ConfigLoggingBackend(conf *viper.Viper) error {
	if !conf.IsSet("log.out") {
		return nil
	}
	out := conf.GetString("log.out")
	return SetLoggingBackend(out)
}

// SetLoggingLevel TODO
func SetLoggingLevel(level string, debug bool) error {

	if strings.TrimSpace(level) == "" {
		level = DefaultLogLevel
	}
	var logLevel logging.Level
	var err error
	if logLevel, err = logging.LogLevel(level); err != nil {
		return err
	}

	if debug {
		logLevel = logging.DEBUG
	}
	logging.SetLevel(logLevel, DefaultLoggerName)
	return nil
}

// SetLoggingFormat TODO
func SetLoggingFormat(format string) error {
	var formatter logging.Formatter
	var err error
	if formatter, err = logging.NewStringFormatter(format); err != nil {
		return err
	}
	logging.SetFormatter(formatter)
	return nil
}

// SetLoggingBackend TODO
func SetLoggingBackend(out string) error {
	var o io.Writer
	switch out {
	case "stdout":
		o = os.Stdout
	case "stderr", "":
		o = os.Stderr
	default:
		f, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)

		if err != nil {
			return err
		}

		o = f
	}

	backend := logging.NewLogBackend(o, "", 0)
	logging.SetBackend(backend)
	return nil
}
