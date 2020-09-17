package cli

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

// CLI holds the context for the command line.
type CLI struct {
	root  *cobra.Command
	serve struct {
		logLevel ZapLevel
		devMode  bool
		otelAddr string
	}
	help bool
}

// Wrap zapcore.Level to implement github.com/spf13/pflag.Value interface.
type ZapLevel struct {
	zapcore.Level
}

// Method Type is part of github.com/spf13/pflag.Value interface.
func (lvl ZapLevel) Type() string {
	return "zapcore.Level"
}

// Create a CLI to run.
func New(version string) *CLI {
	cli := &CLI{}

	// Setup root command.
	cli.root = newCommand()
	cli.root.Version = version
	cli.root.Use = "cloudnative COMMAND"
	cli.root.Short = "Cloud native service example integrated with opentelemetry."
	cli.root.PersistentFlags().BoolVar(&cli.help, "help", false, "Print help and exit")

	// Setup serve command.
	serve := newCommand()
	serve.Use = "serve"
	serve.Short = "Start cloud native servers"
	serve.Run = cli.Serve
	serve.Flags().VarP(&cli.serve.logLevel, "log-level", "l", "Set the logging level")
	serve.Flags().StringVar(&cli.serve.otelAddr, "otel-addr", "", "Set the open telemetry collector's address")
	cli.root.AddCommand(serve)

	return cli
}

// Run CLI.
func (cli *CLI) Run() error {
	return cli.root.Execute()
}

// Create an empty command prefilled with defaults.
func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		DisableAutoGenTag:     true,
		DisableFlagsInUseLine: true,
		SilenceErrors:         true,
		SilenceUsage:          true,
	}
	return cmd
}
