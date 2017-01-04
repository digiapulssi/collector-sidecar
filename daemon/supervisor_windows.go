package daemon

import (
	"github.com/alexbrainman/ps"
	"github.com/alexbrainman/ps/winapi"
	"os/exec"
	"sync"
)

type WindowsSupervisor struct {
	job      *ps.JobObject
	monitor  *ps.JobObjectMonitor
	stopChan chan bool
	events   chan SupervisorEvent
	wg       sync.WaitGroup
	handles  map[int]*SupervisorProcessHandle

	sync.Mutex
}

func NewSupervisor(events chan SupervisorEvent) (Supervisor, error) {
	job, err := ps.CreateJobObject("sidecar")
	if err != nil {
		return nil, err
	}

	monitor, err := job.Monitor()
	if err != nil {
		return nil, err
	}

	return &WindowsSupervisor{
		job:      job,
		monitor:  monitor,
		stopChan: make(chan bool, 1),
		events:   events,
		handles:  make(map[int]*SupervisorProcessHandle),
	}, nil
}

func (s *WindowsSupervisor) Start() error {
	s.wg.Add(1)

	go func() {
		log.Print("starting supervisor loop")

		cancel := false
		for {
			if cancel {
				break
			}

			event, o, err := s.monitor.GetEvent()

			// Check if the go routine should be stopped before checking for error.
			// If we are stopping we just ignore the error.
			select {
			case stop := <-s.stopChan:
				cancel = stop
				continue
			default:
			}

			if err != nil {
				log.Printf("error getting event: %v", err)
				continue
			}

			switch event {
			case winapi.JOB_OBJECT_MSG_NEW_PROCESS:
				log.Printf("SVS new process <%d> started", o)
			case winapi.JOB_OBJECT_MSG_EXIT_PROCESS:
				log.Printf("SVS process <%d> stopped", o)
				s.onStop(int(o))
			case winapi.JOB_OBJECT_MSG_ABNORMAL_EXIT_PROCESS:
				log.Printf("SVS process <%d> abnormal stopped", o)
				s.onStop(int(o))
			default:
				log.Printf("SVS unhandled event <%d>", event)
			}
		}

		log.Print("stopping supervisor loop")
		s.wg.Done()
	}()

	s.job.AddCurrentProcess()

	return nil
}

func (s *WindowsSupervisor) onStop(pid int) {
	handle, ok := s.handles[pid]
	if ok {
		if handle != nil {
			s.events <- SupervisorEvent{Pid: handle.Pid, Type: SupStop}
		}
		s.Lock()
		delete(s.handles, pid)
		s.Unlock()
	}
}

func (s *WindowsSupervisor) Stop() error {
	s.stopChan <- true

	// Close the monitor to unblock the go routine which is blocking on monitor.GetEvent()
	if err := s.monitor.Close(); err != nil {
		return err
	}

	s.wg.Wait()

	if err := s.job.Close(); err != nil {
		return err
	}

	return nil
}

func (s *WindowsSupervisor) StartCommand(cmd *exec.Cmd) (*SupervisorProcessHandle, error) {
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("error waiting for process: %v", err)
		}
	}()

	handle := &SupervisorProcessHandle{
		Pid:          cmd.Process.Pid,
		Cmd:          cmd,
	}

	s.Lock()
	s.handles[handle.Pid] = handle
	s.Unlock()

	// We call the OnStart handler here instead of the job monitor loop because the JOB_OBJECT_MSG_NEW_PROCESS
	// event fired before we registered the handle in the handles map.
	s.events <- SupervisorEvent{Pid: handle.Pid, Type: SupStart}

	return handle, nil
}

func (s *WindowsSupervisor) StopCommand(handle *SupervisorProcessHandle) error {
	if handle.Cmd.Process != nil {
		err := handle.Cmd.Process.Kill()
		if err != nil {
			log.Printf("failed to kill process: %v", err)
			return err
		}
	}

	return nil
}
