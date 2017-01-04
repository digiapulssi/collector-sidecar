// +build linux darwin

package daemon

import (
	"sync"
	"os/exec"
)

type UnixSupervisor struct {
	stopChan chan bool
	wg       sync.WaitGroup
}

func NewSupervisor() (Supervisor, error) {
	return &UnixSupervisor{
		stopChan: make(chan bool),
	}, nil
}

func (s *UnixSupervisor) Start() error {
	return nil
}

func (s *UnixSupervisor) Stop() error {
	s.stopChan <- true

	s.wg.Wait()

	return nil
}

func (s *UnixSupervisor) RunCommand(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("error waiting for process: %v", err)
		}
	}()

	return nil
}
