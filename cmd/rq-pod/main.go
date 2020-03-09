package main

import (
	cmd "github.com/cfhamlet/os-rq-pod/cmd/rq-pod/command"
	"github.com/cfhamlet/os-rq-pod/pkg/command"
)

func main() {
	command.Execute(cmd.Root)
}
