package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LeonidVas/tarantool-supervisor/service/api/supervisorhttp"
	"github.com/LeonidVas/tarantool-supervisor/service/supervisor"
)

// args describes the parsed arguments.
type args struct {
	CfgPath string
	Host    string
}

// parseArgs returns the parsed arguments.
func parseArgs() *args {
	var args args
	flag.StringVar(&args.CfgPath, "cfg", "cfg.json",
		"path to Supervisor config.")
	flag.StringVar(&args.Host, "host", "127.0.0.1:8080",
		"address to start the HTTP server(addr:port).")
	flag.Parse()

	return &args
}

// handleSigChld handles the SIGCHLD signal.
func handleSigChld(sv *supervisor.Supervisor) {
	// Get PID of the terminated Instance.
	pid, err := syscall.Wait4(-1, nil, syscall.WNOHANG, nil)
	if err != nil {
		// If the error "no child processes" (syscall.Errno(10)) -
		// this means that the process was stopped as planned, and
		// the wait () function has already been called.
		if !errors.Is(err, syscall.Errno(10)) {
			log.Printf("Can't get the PID of a process by using \"Wait4\". Error: \"%v\"",
				err)
		}
		return
	}

	// If the isRestart Instance flag is true, try to restart the Instance.
	if id, err := sv.RestartAfterTermInstance(pid); err != nil {
		log.Printf("Can't restart the Instance. Old PID : %v. ID: %v. Error: \"%v\"",
			pid, id, err)
	} else {
		log.Printf("The Instance has been restarted. ID: %v", id)
	}
}

// gracefulTermination terminates the service correctly.
func gracefulTermination(sv *supervisor.Supervisor, srv *http.Server, done chan bool) {
	// First of all, shut down the HTTP server to avoid reciving a new request.
	timeout := 200 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: \"%v\"", err)
	}
	cancel()

	// And now stop all running Instances.
	sv.StopAllInstance()

	done <- true
}

// startSignalHandling adds "SIGTERM" and "SIGINT" signal handling to
// terminate gracefully and "SIGCHLD" to restart Instances.
func startSignalHandling(sv *supervisor.Supervisor, srv *http.Server, done chan bool) {
	// The handling of the "SIGTERM" and "SIGINT" signals
	// will be used to terminate gracefully.
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGINT, syscall.SIGTERM)

	// The handling of the "SIGCHLD" will be used to restart Instances.
	sigChld := make(chan os.Signal, 1)
	signal.Notify(sigChld, syscall.SIGCHLD)

	// Start signals handling.
	go func() {
		for {
			select {
			case _ = <-sigTerm:
				gracefulTermination(sv, srv, done)
			case _ = <-sigChld:
				handleSigChld(sv)
			}
		}
	}()
}

func main() {
	// Get config.
	args := parseArgs()
	cfg, err := supervisor.ParseCfg(args.CfgPath)
	if err != nil {
		log.Fatalf("Can't parse a config: %v", err)
	}

	// Create Supervisor.
	sv := supervisor.NewSupervisor(cfg)

	// Prepare HTTP server.
	svHandler := supervisorhttp.NewSupervisorHandler(sv)
	http.Handle("/instance", svHandler)
	srv := &http.Server{
		Addr: args.Host,
	}

	// Start signal processing.
	done := make(chan bool, 1)
	startSignalHandling(sv, srv, done)

	// Start HTTP server.
	if err = srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Can't start HTTP server")
	}

	<-done
	log.Print("The service has been terminated.")
}
