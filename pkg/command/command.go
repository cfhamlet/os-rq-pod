package command

import (
	"fmt"
	"os"

	"github.com/cfhamlet/os-rq-pod/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRootCommand TODO
func NewRootCommand(program string) *cobra.Command {
	return &cobra.Command{
		Use:   program,
		Short: program + " command line tool",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				if err := cmd.Help(); err != nil {
					panic(err)
				}
				os.Exit(0)
			}
		},
	}
}

// NewVersionCommand TODO
func NewVersionCommand() *cobra.Command {
	verbose := false
	command := &cobra.Command{
		Use:   "version",
		Short: "show version info",
		Run: func(cmd *cobra.Command, args []string) {
			v := version.Version()
			if verbose {
				v = version.VerbosInfo()
			}
			fmt.Fprintln(os.Stderr, v)
		},
	}
	command.Flags().BoolVarP(&verbose, "verbose", "v", false, "show verbose version info")
	return command
}

// NewRunCommand TODO
func NewRunCommand(program string, run RunWithConfig) *cobra.Command {
	conf := viper.New()
	command := &cobra.Command{
		Use:   "run",
		Short: fmt.Sprintf("run %s server", program),
		Run: func(cmd *cobra.Command, args []string) {
			run(conf)
		},
	}

	var cfgFile string
	command.Flags().StringVarP(&cfgFile, "conf", "c", "", "config file")
	if err := command.MarkFlagRequired("conf"); err != nil {
		panic(fmt.Errorf("%w", err))
	}
	if err := conf.BindPFlag("configfile", command.Flags().Lookup("conf")); err != nil {
		panic(fmt.Errorf("%w", err))
	}

	command.Flags().Bool("debug", false, "enable debug model")
	if err := conf.BindPFlag("debug", command.Flags().Lookup("debug")); err != nil {
		panic(fmt.Errorf("%w", err))
	}

	var logLevel string
	command.Flags().StringVarP(&logLevel, "loglevel", "L", "info", "log level")
	if err := conf.BindPFlag("log.level", command.Flags().Lookup("loglevel")); err != nil {
		panic(fmt.Errorf("%w", err))
	}
	return command
}

// Execute TODO
func Execute(root *cobra.Command) {
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// RunWithConfig TODO
type RunWithConfig func(*viper.Viper)
