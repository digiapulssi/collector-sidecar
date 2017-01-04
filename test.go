package main

import (
	"github.com/Graylog2/collector-sidecar/daemon"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

//const filebeat = "/home/bernd/local/filebeat-5.0.0-linux-x86_64/filebeat"
//const config = "/dev/shm/sidecar/generated-filebeat.yml"

const filebeat = "F:\\sidecar\\filebeat.exe"
const config = "F:\\sidecar\\generated-filebeat.yml"

func main() {
	var supervisor daemon.Supervisor

	events := make(chan daemon.SupervisorEvent, 128)

	supervisor, err := daemon.NewSupervisor(events)
	if err != nil {
		log.Printf("error creating supervisor: %v", err)
		os.Exit(1)
	}

	if err := supervisor.Start(); err != nil {
		log.Printf("error starting supervisor: %v", err)
		os.Exit(1)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	startProcess(supervisor)
	startProcess(supervisor)

MAINLOOP:
	for {
		select {
		case sig := <-signals:
			log.Printf("got signal <%v>", sig)
			break MAINLOOP
		case event := <-events:
			log.Printf("received supervisor event <%#v>", event)
		}
	}

	if err := supervisor.Stop(); err != nil {
		log.Printf("error stopping supervisor: %v", err)
	}

	log.Print("STOPPED!")
}

func startProcess(supervisor daemon.Supervisor) {
	cmd := exec.Command(filebeat, "-c", config)

	_, err := supervisor.StartCommand(cmd)
	if err != nil {
		log.Printf("error starting command <%v>: %v", cmd.Args, err)
		return
	}
}

func isProcessRunning2(pid int) bool {
	process, err := os.FindProcess(pid)

	if err != nil {
		return false
	}

	// On Unix systems, FindProcess always succeeds and returns a Process for the given pid,
	// regardless of whether the process exists. That's why we use signal 0 to check if the process exists.
	if runtime.GOOS != "windows" {
		return process.Signal(syscall.Signal(0)) == nil
	}

	return true
}
