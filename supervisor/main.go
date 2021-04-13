package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tarantool/tvisor/supervisor/api/supervisorhttp"
	"github.com/tarantool/tvisor/supervisor/core"
)

// args describes the parsed arguments.
type args struct {
	//CfgPath - path to Tvisor config.
	CfgPath string
	// Addr - address to start the HTTP server(host:port).
	Addr string
}

// parseArgs returns the parsed arguments.
func parseArgs() *args {
	var args args
	flag.StringVar(&args.CfgPath, "cfg", "cfg.json",
		"path to Tvisor config.")
	flag.StringVar(&args.Addr, "addr", "127.0.0.1:8080",
		"address to start the HTTP server(host:port).")
	flag.Parse()

	return &args
}

// parseCfg parses the Tvisor JSON config.
func parseCfg(path string) (*core.Cfg, error) {
	// Check is the file exists.
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	// Open the file.
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	// Set defaults.
	cfg := core.Cfg{
		InstancesDir: "/etc/tarantool/tvisor/instances",
		TermTimeout:  30,
	}

	// Read and parse config.
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(byteValue, &cfg); err != nil {
		return nil, err
	}

	// In the config, the time is indicated in seconds. Convert the value.
	cfg.TermTimeout = cfg.TermTimeout * time.Second

	return &cfg, nil
}

// handleZombie handles zombie child processes, if any, in a non-blocking style.
func handleZombie(sv *core.Supervisor) {
	// Get PID of the terminated Instance.
	pid, err := syscall.Wait4(-1, nil, syscall.WNOHANG, nil)

	// If the error "no child processes" (syscall.Errno(10)) or pid == 0 -
	// this means that the process was stopped as planned, and the wait ()
	// function has already been called.
	if pid == 0 || (err != nil && errors.Is(err, syscall.Errno(10))) {
		return
	} else if err != nil {
		log.Printf(`Can't get the PID of a process by using "Wait4". Error: "%v"`,
			err)
		return
	}

	// If the Restartable Instance flag is true, try to restart the Instance.
	if id, err := sv.RestartAfterTermInstance(pid); err != nil {
		log.Printf(`Can't restart the Instance. Old PID : %v. ID: %v. Error: "%v"`,
			pid, id, err)
	} else {
		log.Printf("The Instance has been restarted. ID: %v", id)
	}
}

// terminateGracefully terminates the service correctly.
func terminateGracefully(sv *core.Supervisor, srv *http.Server,
	timeout time.Duration, done chan bool) {
	// First of all, shut down the HTTP server to avoid reciving a new request.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf(`HTTP server shutdown error: "%v"`, err)
	}
	cancel()

	// And now stop all running Instances.
	sv.StopAllInstances()

	done <- true
}

// startSignalHandling adds "SIGTERM" and "SIGINT" signal handling to
// terminate gracefully and "SIGCHLD" to restart Instances.
func startSignalHandling(sv *core.Supervisor, srv *http.Server,
	termTimeout time.Duration, done chan bool) {
	// The handling of the "SIGTERM" and "SIGINT" signals
	// will be used to terminate gracefully.
	sigTermChan := make(chan os.Signal, 1)
	signal.Notify(sigTermChan, syscall.SIGINT, syscall.SIGTERM)

	// The handling of the "SIGCHLD" will be used to restart Instances.
	sigChldChan := make(chan os.Signal, 1)
	signal.Notify(sigChldChan, syscall.SIGCHLD)

	// Start signals handling.
	go func() {
		for {
			select {
			case _ = <-sigTermChan:
				terminateGracefully(sv, srv, termTimeout, done)
			case _ = <-sigChldChan:
				handleZombie(sv)
			case <-time.After(20 * time.Second):
				// According to
				// https://www.gnu.org/software/libc/manual/html_node/Merged-Signals.html,
				// if at the moment we handle the SIGHILD signal, we receive
				// two or more SIGHILD signals, only one of them will be handled.
				// This doesn't seem like a common case for us, but we must have
				// some kind of guard for this case. Let's just check periodically
				// if we have some zombie process pending processing.
				handleZombie(sv)
			}
		}
	}()
}

func main() {
	// Get config.
	args := parseArgs()
	cfg, err := parseCfg(args.CfgPath)
	if err != nil {
		log.Fatalf("Can't parse a config: %v", err)
	}

	// Create Supervisor.
	sv := core.NewSupervisor(cfg)

	// Prepare HTTP server.
	svHandler := supervisorhttp.NewSupervisorHandler(sv)
	http.Handle("/instance", svHandler)
	srv := &http.Server{
		Addr: args.Addr,
	}

	// We will use the instance completion timeout multiplied
	// by 5 as the service termination timeout.
	serviceTermTimeout := cfg.TermTimeout * 5
	// Start signal processing.
	done := make(chan bool, 1)
	startSignalHandling(sv, srv, serviceTermTimeout, done)

	// Start HTTP server.
	if err = srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Can't start HTTP server")
	}

	<-done
	log.Print("The service has been terminated.")
}
