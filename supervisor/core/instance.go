package core

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Instance states.
const (
	stateTerminated = "terminated"
	stateRunning    = "running"
)

// Instance describes a running process.
type Instance struct {
	// Name is the name of the Instance.
	Name string
	// Cmd represents an external command being prepared or run.
	Cmd *exec.Cmd
	// Restartable indicates whether to restart the instance in
	// case of failure or not.
	Restartable bool
	// Env describes the environment settled by a client.
	Env []string
	// mutex is used to prevent prevent multiple goroutines
	// from trying to stop an instance at the same time.
	mutex sync.Mutex
	// done channel used to wait for a process termination.
	done chan error
}

// InstanceStatus describes the status of the Instance.
type InstanceStatus struct {
	// Name is the name of the Instance.
	Name string `json:"name"`
	// State describes the state of the Instance.
	// Available values: see Instance states.
	State string `json:"state"`
	// Pid is a process ID.
	Pid int `json:"pid"`
	// Restartable indicates whether to restart the instance in
	// case of failure or not.
	Restartable bool `json:"restartable"`
	// Env describes the environment settled by a client.
	Env []string `json:"env"`
}

// NewInstance creates an Instance.
func NewInstance(name string, cmd *exec.Cmd, env []string, restart bool) *Instance {
	return &Instance{Name: name, Cmd: cmd, Env: env, Restartable: restart}
}

// IsAlive verifies that the Instance is alive by sending a "0" signal.
func (inst *Instance) IsAlive() bool {
	return inst.Cmd.Process.Signal(syscall.Signal(0)) == nil
}

// Start runs the Instatnce.
func (inst *Instance) Start() error {
	return inst.Cmd.Start()
}

// Restart restarts the Instance.
func (inst *Instance) Restart() error {
	// Seems like to restart and to stop the same Instance
	// at the same time is a bad idea. Let's lock the mutex.
	inst.mutex.Lock()
	defer inst.mutex.Unlock()

	// Restart the Instance.
	cmd := exec.Command(inst.Cmd.Path)
	cmd.Env = append(os.Environ(), inst.Env...)
	inst.Cmd = cmd
	return inst.Cmd.Start()
}

// Stop terminates the Instance.
//
// timeout - the time that was provided to the process
// to terminate correctly befor the "SIGKILL" signal is used.
//
// force - if force is "true" the "SIGKILL" signal will be
// sent to the process in case of using "SIGINT" doesn't
// terminate the process.
func (inst *Instance) Stop(timeout time.Duration, force bool) error {
	// Attempt to stop the same process from several goroutines
	// at the same time doesn't seem like a good idea. To
	// avoid this, we'll use a mutex during process termination.
	inst.mutex.Lock()
	defer inst.mutex.Unlock()

	// Instance shouldnёt be restarted if a stop command was received for it
	inst.Restartable = false

	// Check if the process is running by sending a signal "0".
	if !inst.IsAlive() {
		return nil
	}

	// First of all start wait for the process to terminate.
	// The inst.done channel is initialized on the first
	// attempt to terminate the Instance.
	if inst.done == nil {
		inst.done = make(chan error, 1)
		go func() {
			inst.done <- inst.Cmd.Wait()
		}()
	}

	// Trying to terminate the process by using a "SIGINT" signal.
	// In case of failure and if the force is "true", a "SIGKILL"
	// signal will be used.
	if err := inst.Cmd.Process.Signal(os.Interrupt); err != nil {
		return err
	}

	select {
	case <-time.After(timeout):
		if !force {
			return errors.New("The process couldn't be terminated correctly.")
		}
		// Send "SIGKILL" signal
		if err := inst.Cmd.Process.Kill(); err != nil {
			return err
		} else {
			// Wait for the process to terminate.
			_ = <-inst.done
			return nil
		}
	case err := <-inst.done:
		return err
	}
}

// Status returns the current status of the Instance.
func (inst *Instance) Status() *InstanceStatus {
	res := InstanceStatus{
		Name:        inst.Name,
		Pid:         inst.Cmd.Process.Pid,
		Restartable: inst.Restartable,
		Env:         inst.Env,
	}
	if inst.IsAlive() {
		res.State = stateRunning
	} else {
		res.State = stateTerminated
	}
	return &res
}
