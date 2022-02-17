package poetryinstall

import (
	"fmt"
	"os"
	"path/filepath"
)

// PythonPathProcess implements the Executable interface.
type PythonPathProcess struct {
}

// NewPythonPathProcess creates an instance of the PythonPathProcess.
func NewPythonPathProcess() PythonPathProcess {
	return PythonPathProcess{}
}

// Execute locates the Python path (AKA site-packages directory) within the poetry targetLayerPath.
func (p PythonPathProcess) Execute(venvDir string) (string, error) {
	// Poetry does not have a built in way to return the site-packages directory
	// But we know the structure underneath the virtual env dir looks as follows:
	// virtual-env-dir/pythonX.Y/lib/site-packages
	// So we can find site-packages by traversing the known directory structure
	libDir := filepath.Join(venvDir, "lib")
	libEntries, err := os.ReadDir(libDir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: '%s':\nerror: %w", libDir, err)
	}

	if len(libEntries) > 1 {
		return "", fmt.Errorf("expected one directory and zero files in directory: '%s' - found multiple", libDir)
	}

	pythonDir := libEntries[0]

	pythonDirPath := filepath.Join(libDir, pythonDir.Name())
	if !pythonDir.IsDir() {
		return "", fmt.Errorf("expected a directory at: '%s'", pythonDirPath)
	}

	entries, err := os.ReadDir(pythonDirPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: '%s':\nerror: %w", pythonDirPath, err)
	}

	if len(entries) > 1 {
		return "", fmt.Errorf("expected one directory and zero files in directory: '%s' - found multiple", pythonDirPath)
	}

	sitePackagesDir := entries[0]

	sitePackagesPath := filepath.Join(pythonDirPath, "site-packages")

	if sitePackagesDir.Name() != "site-packages" {
		return "", fmt.Errorf(`expected "site-packages" directory at: '%s', found: %s`, sitePackagesPath, sitePackagesDir.Name())
	}

	if !sitePackagesDir.IsDir() {
		return "", fmt.Errorf("expected a directory at: '%s'", sitePackagesPath)
	}

	return filepath.Join(pythonDirPath, "site-packages"), nil
}
