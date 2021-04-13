// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// can be overwritten by GOEXE
var goExe = "go"

// can be overwritten by TVISOREXE
var tvExe = "tvisor"

var goPackageName = "github.com/tarantool/tvisor/supervisor"
var packagePath = "./supervisor"

func getBuildEnv() map[string]string {
	var err error

	var curDir string
	var gitTag string
	var gitCommit string

	if curDir, err = os.Getwd(); err != nil {
		fmt.Printf("Failed to get current directory: %s\n", err)
	}

	if _, err := exec.LookPath("git"); err == nil {
		gitTag, _ = sh.Output("git", "describe", "--tags")
		gitCommit, _ = sh.Output("git", "rev-parse", "--short", "HEAD")

	}

	versionLabel := os.Getenv("VERSION_LABEL")

	return map[string]string{
		"PACKAGE":       goPackageName,
		"GIT_TAG":       gitTag,
		"GIT_COMMIT":    gitCommit,
		"VERSION_LABEL": versionLabel,
		"PWD":           curDir,
	}
}

var asmflags = "all=-trimpath=${PWD}"
var gcflags = "all=-trimpath=${PWD}"

func init() {
	var err error

	if specifiedGoExe := os.Getenv("GOEXE"); specifiedGoExe != "" {
		goExe = specifiedGoExe
	}

	if specifiedTvExe := os.Getenv("TVEXE"); specifiedTvExe != "" {
		tvExe = specifiedTvExe
	} else {
		if tvExe, err = filepath.Abs(tvExe); err != nil {
			panic(err)
		}
	}

	// We want to use Go 1.11 modules even if the source lives inside GOPATH.
	// The default is "auto".
	os.Setenv("GO111MODULE", "on")
}

// Run go vet and flake8
func Lint() error {
	fmt.Println("Running go vet...")
	if err := sh.RunV(goExe, "vet", packagePath); err != nil {
		return err
	}

	return nil
}

// Run unit tests
func Unit() error {
	fmt.Println("Running unit tests...")

	if mg.Verbose() {
		return sh.RunV(goExe, "test", "-v", "./supervisor/...")
	} else {
		return sh.RunV(goExe, "test", "./supervisor/...")
	}
}

// Run all tests
func Test() {
	mg.SerialDeps(Lint, Unit)
}

// Build tvisor executable
func Build() error {
	var err error

	fmt.Println("Building...")

	err = sh.RunWith(
		getBuildEnv(), goExe, "build",
		"-o", tvExe,
		"-asmflags", asmflags,
		"-gcflags", gcflags,
		packagePath,
	)

	if err != nil {
		return fmt.Errorf("Failed to build tvisor executable: %s", err)
	}

	return nil
}

// Clean up after yourself
func Clean() {
	fmt.Println("Cleaning...")

	os.RemoveAll(tvExe)
}
