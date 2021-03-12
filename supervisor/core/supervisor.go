/*
Supervisor provides the ability to manage Instances.
*/
package core

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
	// instMapMutex is used to work with the map of Instances.
	instMapMutex sync.RWMutex
	// instancesById is a map of running Instances.
	instancesById map[int]*Instance
	// cfg is a pointer to a Supervisor config.
	cfg *Cfg
	// lastId is an id of the last running Instance.
	lastId int
}

// NewSupervisor creates a Supervisor.
func NewSupervisor(cfg *Cfg) *Supervisor {
	sv := new(Supervisor)
	sv.instancesById = make(map[int]*Instance)
	sv.cfg = cfg
	return sv
}

// addInstance adds the Instance to the Supervisor map.
// Return ID of the Instance.
func (sv *Supervisor) addInstance(inst *Instance) int {
	sv.instMapMutex.Lock()
	defer sv.instMapMutex.Unlock()
	sv.lastId++
	id := sv.lastId
	sv.instancesById[id] = inst
	return id
}

// deleteInstance removes the Instance from the Supervisor map.
func (sv *Supervisor) deleteInstance(id int) {
	sv.instMapMutex.Lock()
	defer sv.instMapMutex.Unlock()
	delete(sv.instancesById, id)
}

// getInstance return a pointer to the Instance by id.
func (sv *Supervisor) getInstance(id int) *Instance {
	sv.instMapMutex.RLock()
	defer sv.instMapMutex.RUnlock()
	inst := sv.instancesById[id]
	return inst
}

// getInstanceByPid return an ID and a pointer ro the Instance by pid.
func (sv *Supervisor) getInstanceByPid(pid int) (int, *Instance) {
	// Seems like this shouldn't be a popular method
	// and a full scan can be used.
	sv.instMapMutex.RLock()
	defer sv.instMapMutex.RUnlock()
	for id, inst := range sv.instancesById {
		if inst.Cmd.Process.Pid == pid {
			return id, inst
		}
	}
	return 0, nil
}

// StartInstance starts a new Instance with the specified parameters.
// On fail returns 0, error.
func (sv *Supervisor) StartInstance(name string, env []string, restartable bool) (int, error) {
	// When Supervisor is terminating, we will lock "termMutex"
	// to prevent new instances from starting during Supervisor termination.
	sv.termMutex.RLock()
	defer sv.termMutex.RUnlock()

	// Form the path to the instance and check is it exists.
	if name == "" {
		return 0, errors.New(`The instance name is empty.`)
	}
	instPath := path.Join(sv.cfg.InstancesDir, name+".lua")
	if _, err := exec.LookPath(instPath); err != nil {
		return 0, err
	}

	// Start an Instance.
	cmd := exec.Command(instPath)
	cmd.Env = append(os.Environ(), env...)
	inst := NewInstance(name, cmd, env, restartable)
	if err := inst.Start(); err != nil {
		return 0, err
	}

	return sv.addInstance(inst), nil
}

// RestartAfterTermInstance should be used to restart an instance in case
// of an unexpected termination of the instance.
// First of all, the method checks if the instance is alive. In this case,
// the method returns the current Instance ID with an error.
// If the Instance has been terminated and Instance.Restartable is true,
// then the Instance will be restarted.
// On success returns the Instance ID (0 invalid).
func (sv *Supervisor) RestartAfterTermInstance(pid int) (int, error) {
	// Find an Instance.
	id, inst := sv.getInstanceByPid(pid)
	if inst == nil {
		return 0, errors.New("Unknown Instance.")
	}

	// We don't want to restart the Instance at the same time
	// as StopAllInstances is running.
	sv.termMutex.RLock()
	defer sv.termMutex.RUnlock()

	// If the Instance is alive something went wrong.
	if inst.IsAlive() {
		return id, errors.New("Instance is alive.")
	}

	if !inst.Restartable {
		return id, errors.New("The Restart Instance flag is false.")
	}

	if err := inst.Restart(); err != nil {
		return id, err
	}

	return id, nil
}

// StopInstance terminate an Instance by ID.
func (sv *Supervisor) StopInstance(id int, force bool) error {
	// When Supervisor is terminating, we will lock "termMutex"
	// to prevent an instance termination from several places
	// (see StopAllInstances).
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

// StopAllInstances terminate all Instances managed by the Supervisor.
func (sv *Supervisor) StopAllInstances() {
	// Disable start / stop Instances.
	sv.termMutex.Lock()

	// Start termination of all Instances.
	var wg sync.WaitGroup
	// We shouldn't delete instances from the map
	// until the iteration is done.
	sv.instMapMutex.RLock()
	defer sv.termMutex.Unlock()
	for id, inst := range sv.instancesById {
		wg.Add(1)
		go func(id int, inst *Instance) {
			inst.Stop(sv.cfg.TermTimeout, true)
			sv.deleteInstance(id)
			wg.Done()
		}(id, inst)
	}
	// Here you need to Unlock the mutex, because while it is Lock,
	// no one goroutines created in the for loop will be done
	// (see the "deleteInstance" method).
	sv.instMapMutex.RUnlock()
	// Wait for the end of the termination process.
	wg.Wait()

}

// GetInstanceStatus returns the current status of the Instance.
// On failure, returns an empty string and error.
func (sv *Supervisor) GetInstanceStatus(id int) (*InstanceStatus, error) {
	inst := sv.getInstance(id)

	if inst == nil {
		return nil, errors.New("Unknown instance with id " + strconv.Itoa(id))
	}

	return inst.Status(), nil
}

// ListInstances returns a list of running instances.
func (sv *Supervisor) ListInstances() map[string]*InstanceStatus {
	sv.instMapMutex.RLock()
	defer sv.instMapMutex.RUnlock()
	instsMap := make(map[string]*InstanceStatus)
	for id, inst := range sv.instancesById {
		// For following serialization to JSON, a string is used as
		// a key instead of an int.
		instsMap[strconv.Itoa(id)] = inst.Status()
	}

	return instsMap
}
