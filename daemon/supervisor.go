package daemon

import "os/exec"

const (
	SupStart = iota
	SupStop
)

type Supervisor interface {
	Start() error
	Stop() error
	StartCommand(cmd *exec.Cmd) (*SupervisorProcessHandle, error)
	StopCommand(handle *SupervisorProcessHandle) error
}

type SupervisorProcessHandle struct {
	Pid    int
	Cmd    *exec.Cmd
}

type SupervisorEvent struct {
	Type int
	Pid  int
}
