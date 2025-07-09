package poetryinstall

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface Executable --output fakes/executable.go

// Executable defines the interface for invoking an executable.
type Executable interface {
	Execute(pexec.Execution) error
}

// PoetryInstallProcess implements the InstallProcess interface.
type PoetryInstallProcess struct {
	executable Executable
	logger     scribe.Emitter
}

// NewPoetryInstallProcess creates an instance of the PoetryInstallProcess given an Executable.
func NewPoetryInstallProcess(executable Executable, logger scribe.Emitter) PoetryInstallProcess {
	return PoetryInstallProcess{
		executable: executable,
		logger:     logger,
	}
}

// Execute installs the poetry dependencies from workingDir/pyproject.toml into
// a virtual env in the targetPath.
func (p PoetryInstallProcess) Execute(workingDir, targetPath, cachePath string) (string, error) {
	installOnly, exists := os.LookupEnv("BP_POETRY_INSTALL_ONLY")
	if !exists {
		installOnly = "main"
	}

	args := []string{"install", "--only", installOnly}

	env := append(
		os.Environ(),
		fmt.Sprintf("POETRY_CACHE_DIR=%s", cachePath),
		fmt.Sprintf("POETRY_VIRTUALENVS_PATH=%s", targetPath),
	)

	p.logger.Subprocess(fmt.Sprintf("Running 'POETRY_CACHE_DIR=%s POETRY_VIRTUALENVS_PATH=%s poetry %s'", cachePath, targetPath, strings.Join(args, " ")))

	err := p.executable.Execute(pexec.Execution{
		Args:   args,
		Env:    env,
		Dir:    workingDir,
		Stdout: p.logger.ActionWriter,
		Stderr: p.logger.ActionWriter,
	})
	if err != nil {
		return "", fmt.Errorf("poetry install failed:\nerror: %w", err)
	}

	return p.findVenvDir(workingDir, targetPath, cachePath)
}

func (p PoetryInstallProcess) findVenvDir(workingDir, targetPath, cachePath string) (string, error) {
	env := append(
		os.Environ(),
		fmt.Sprintf("POETRY_CACHE_DIR=%s", cachePath),
		fmt.Sprintf("POETRY_VIRTUALENVS_PATH=%s", targetPath),
	)

	args := []string{"env", "info", "--path"}

	outBuffer := bytes.NewBuffer(nil)
	errBuffer := bytes.NewBuffer(nil)
	err := p.executable.Execute(pexec.Execution{
		Args:   args,
		Env:    env,
		Dir:    workingDir,
		Stdout: outBuffer,
		Stderr: errBuffer,
	})
	if err != nil {
		return "", fmt.Errorf("failed to find virtual env directory:\n%s\n%s\nerror: %w", outBuffer, errBuffer, err)
	}

	return filepath.Clean(strings.TrimSpace(outBuffer.String())), nil
}
