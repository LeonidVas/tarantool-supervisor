/*
Supervisor provides the ability to manage Instances using the HTTP API.
*/
package supervisor

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
)

// Supervisor stores the information about started Instances.
type Supervisor struct {
	// termMutex is used to prevent Instances from
	// start / restart / stop during Supervisor termination.
	termMutex sync.RWMutex
	// instMutex is used to work with the map of Instances.
	instMutex sync.RWMutex
	// instaces is a map of running Instances.
	instances map[int]*Instance
	// cfg is a pointer to a Supervisor config.
	cfg *Cfg
	// lastId is an id of the last running Instance.
	lastId int
}

// NewSupervisor creates a Supervisor.
func NewSupervisor(cfg *Cfg) *Supervisor {
	sv := new(Supervisor)
	sv.instances = make(map[int]*Instance)
	sv.cfg = cfg
	return sv
}

// addInstance adds the Instance to the Supervisor map.
// Return ID of the Instance.
func (sv *Supervisor) addInstance(inst *Instance) int {
	sv.instMutex.Lock()
	sv.lastId++
	id := sv.lastId
	sv.instances[id] = inst
	sv.instMutex.Unlock()
	return id
}

// deleteInstance removes the Instance from the Supervisor map.
func (sv *Supervisor) deleteInstance(id int) {
	sv.instMutex.Lock()
	delete(sv.instances, id)
	sv.instMutex.Unlock()
}

// getInstance return a pointer to the Instance by id.
func (sv *Supervisor) getInstance(id int) *Instance {
	sv.instMutex.RLock()
	inst := sv.instances[id]
	sv.instMutex.RUnlock()
	return inst
}

// getInstanceByPid return an ID and a pointer ro the Instance by pid.
func (sv *Supervisor) getInstanceByPid(pid int) (int, *Instance) {
	// Seems like this shouldn't be a popular method
	// and a full scan can be used.
	sv.instMutex.RLock()
	defer sv.instMutex.RUnlock()
	for id, inst := range sv.instances {
		if inst.Cmd.Process.Pid == pid {
			return id, inst
		}
	}
	return 0, nil
}

// StartInstance starts a new Instance with the specified parameters.
// On fail returns 0, error.
func (sv *Supervisor) StartInstance(name string, env []string, isRestart bool) (int, error) {
	// When Supervisor is terminating, we will lock "termMutex"
	// to prevent new instances from starting during Supervisor termination.
	sv.termMutex.RLock()
	defer sv.termMutex.RUnlock()

	// Form the path to the instance and check is it exists.
	instPath := path.Join(sv.cfg.InstancesDir, name+".lua")
	if _, err := exec.LookPath(instPath); err != nil {
		return 0, err
	}

	// Start an Instance.
	cmd := exec.Command(instPath)
	cmd.Env = append(os.Environ(), env...)
	inst := NewInstance(name, cmd, env, isRestart)
	if err := inst.Start(); err != nil {
		return 0, err
	}

	return sv.addInstance(inst), nil
}

// RestartAfterTermInstance checks if the instance is alive.
// If the Instance has been terminated and Instance.IsRestart is true,
// then the Instance will be restarted.
// On success returns the Instance ID (0 invalid).
func (sv *Supervisor) RestartAfterTermInstance(pid int) (int, error) {
	// Find an Instance.
	id, inst := sv.getInstanceByPid(pid)
	if inst == nil {
		return 0, errors.New("Unknown Instance.")
	}

	// We don't want to restart the Instance at the same time
	// as StopAllInstance is running.
	sv.termMutex.RLock()
	defer sv.termMutex.RUnlock()

	// If the Instance is alive something went wrong.
	if inst.IsAlive() {
		return id, errors.New("Instance is alive.")
	}

	if !inst.IsRestart {
		return id, errors.New("The IsRestart Instance flag is false.")
	}

	if err := inst.Restart(); err != nil {
		return id, err
	}

	return id, nil
}

// StopInstance terminate an Instance by id.
func (sv *Supervisor) StopInstance(id int, force bool) error {
	// When Supervisor is terminating, we will lock "termMutex"
	// to prevent an instance termination from several places
	// (see StopAllInstance).
	sv.termMutex.RLock()
	defer sv.termMutex.RUnlock()

	inst := sv.getInstance(id)
	if inst == nil {
		return errors.New("Unknown instance with id " + strconv.Itoa(id))
	}

	if err := inst.Stop(sv.cfg.TermTimeout, force); err != nil {
		return err
	}
	sv.deleteInstance(id)

	return nil
}

// StopAllInstance terminate all Instances managed by the Supervisor.
func (sv *Supervisor) StopAllInstance() {
	// Disable start / stop Instances.
	sv.termMutex.Lock()

	// Start termination of all Instances.
	var wg sync.WaitGroup
	// We don't want to delete instances from the map
	// until the iteration is done.
	sv.instMutex.RLock()
	for id, inst := range sv.instances {
		wg.Add(1)
		go func(id int, inst *Instance) {
			inst.Stop(sv.cfg.TermTimeout, true)
			sv.deleteInstance(id)
			wg.Done()
		}(id, inst)
	}
	sv.instMutex.RUnlock()
	// Wait for the end of the termination process.
	wg.Wait()

	sv.termMutex.Unlock()
}

// StatusInstance returns the current status of the Instance.
// On failure, returns an empty string and error.
func (sv *Supervisor) StatusInstance(id int) (string, error) {
	inst := sv.getInstance(id)

	if inst == nil {
		return "", errors.New("Unknown instance with is " + strconv.Itoa(id))
	}

	return inst.Status(), nil
}

// ListInstances returns a list of running instances.
func (sv *Supervisor) ListInstances() []string {
	sv.instMutex.RLock()
	instsCount := len(sv.instances)
	instsSl := make([]string, 0, instsCount)
	for id, inst := range sv.instances {
		instsSl = append(instsSl, strconv.Itoa(id)+": "+inst.Status())
	}
	sv.instMutex.RUnlock()

	return instsSl
}
