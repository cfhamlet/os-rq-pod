package command

import "github.com/cfhamlet/os-rq-pod/pkg/command"

func init() {
	Root.AddCommand(command.NewVersionCommand())
}
