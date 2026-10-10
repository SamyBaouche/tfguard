package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Version is set at build time: -ldflags="-X main.Version=..."
var Version = "0.1.0"

const errCommandRequired = "command required: scan | version"

// exitError carries a process exit code through Cobra's error return.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}
	// Missing required flags / unknown commands → usage
	return 2
}

// newRootCmd builds the Cobra command tree.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tfguard",
		Short: "Terraform plan risk and policy reviewer",
		Long: `tfguard parses terraform plan JSON, scores change risk,
and evaluates security policies (OPA, Checkov, tfsec).

Commands:
  scan      Analyze a plan and print a risk / policy report
  version   Print the build version`,
		Example: `  tfguard scan --plan plan.json
  tfguard scan --plan plan.json --fail-on DANGER
  tfguard version`,
		// Running with no subcommand shows help and returns an error we map to exit 2.
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd.Help()
			return &exitError{code: 2, msg: errCommandRequired}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.SetHelpTemplate(rootHelpTemplate)
	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newVersionCmd())
	return cmd
}

// rootHelpTemplate is a clearer, sectioned help layout.
const rootHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// execute runs the CLI with args (without the program name) and returns an exit code.
func execute(stdout, stderr io.Writer, args []string) int {
	root := newRootCmd()
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)

	err := root.Execute()
	if err != nil {
		// The root command already printed help for this one; don't repeat it.
		var ee *exitError
		if !(errors.As(err, &ee) && ee.code == 2 && ee.msg == errCommandRequired) {
			fmt.Fprintln(stderr, err)
		}
	}
	return exitCode(err)
}
