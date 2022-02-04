package poetryinstall_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitPoetryInstall(t *testing.T) {
	suite := spec.New("poetryinstall", spec.Report(report.Terminal{}))
	suite("Detect", testDetect)
	suite("Build", testBuild)
	suite("InstallProcess", testInstallProcess)
	suite("PythonPathProcess", testPythonPathProcess)
	suite.Run(t)
}
