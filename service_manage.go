package main

import (
	"fmt"
    "time"

    "golang.org/x/sys/windows/svc"
    "golang.org/x/sys/windows/svc/mgr"
)

type StartCommand struct {}
type StopCommand struct {}
type DebugCommand struct {}

var startCommand StartCommand
var stopCommand StopCommand
var debugCommand DebugCommand

func (x *StartCommand) Execute(args []string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService("zohm")
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()
	err = s.Start("is", "manual-started")
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}
	return nil
}

func (x *StopCommand) Execute(args []string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService("zohm")
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()
	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", svc.Stop, err)
	}
	timeout := time.Now().Add(10 * time.Second)
	for status.State != svc.Stopped {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", svc.Stopped)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil
}

func (x *DebugCommand) Execute(args []string) error {
    runService("zohm", true)
	return nil
}

func init() {
	parser.AddCommand("start",
		"Start service",
		"", &startCommand)

	parser.AddCommand("stop",
		"Stop service",
		"", &stopCommand)

	parser.AddCommand("debug",
		"Debug mode",
		"", &debugCommand)
}
