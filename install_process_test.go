package poetryinstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	poetryinstall "github.com/paketo-buildpacks/poetry-install"
	"github.com/paketo-buildpacks/poetry-install/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testInstallProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		packagesLayerPath string
		cacheLayerPath    string
		workingDir        string
		executable        *fakes.Executable
		buffer            *bytes.Buffer

		executableInvocations []pexec.Execution

		poetryInstallProcess poetryinstall.PoetryInstallProcess
	)

	it.Before(func() {
		var err error
		packagesLayerPath, err = os.MkdirTemp("", "packages")
		Expect(err).NotTo(HaveOccurred())

		cacheLayerPath, err = os.MkdirTemp("", "cache")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "workingdir")
		Expect(err).NotTo(HaveOccurred())

		executable = &fakes.Executable{}

		executableInvocations = []pexec.Execution{}

		executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
			executableInvocations = append(executableInvocations, execution)
			// Various path constructs (like .. and // and whitespace) to validate that we are cleaning the absolute filepath
			// when required
			fmt.Fprintln(execution.Stdout, "//some/path/xyz/../to/some/venv//")
			fmt.Fprintln(execution.Stderr, "stderr output")
			Expect(err).NotTo(HaveOccurred())
			return nil
		}
		buffer = bytes.NewBuffer(nil)

		poetryInstallProcess = poetryinstall.NewPoetryInstallProcess(executable, scribe.NewEmitter(buffer))
	})

	it.After(func() {
		Expect(os.RemoveAll(packagesLayerPath)).To(Succeed())
		Expect(os.RemoveAll(cacheLayerPath)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("Execute", func() {
		it("runs installation", func() {
			venvDir, err := poetryInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(executable.ExecuteCall.CallCount).To(Equal(2))
			Expect(executableInvocations).To(HaveLen(2))

			Expect(executableInvocations[0]).To(MatchFields(IgnoreExtras, Fields{
				"Args": Equal([]string{
					"install", "--only", "main",
				}),
				"Dir": Equal(workingDir),
				"Env": ContainElements([]string{
					fmt.Sprintf("POETRY_VIRTUALENVS_PATH=%s", packagesLayerPath),
					fmt.Sprintf("POETRY_CACHE_DIR=%s", cacheLayerPath),
				}),
			}))

			Expect(executableInvocations[1]).To(MatchFields(IgnoreExtras, Fields{
				"Args": Equal([]string{
					"env", "info", "--path",
				}),
				"Dir": Equal(workingDir),
				"Env": ContainElements([]string{
					fmt.Sprintf("POETRY_VIRTUALENVS_PATH=%s", packagesLayerPath),
					fmt.Sprintf("POETRY_CACHE_DIR=%s", cacheLayerPath),
				}),
			}))

			Expect(venvDir).To(Equal("/some/path/to/some/venv"))
			Expect(buffer.String()).To(ContainLines(
				fmt.Sprintf("    Running 'POETRY_CACHE_DIR=%s POETRY_VIRTUALENVS_PATH=%s poetry install --only main'", cacheLayerPath, packagesLayerPath),
				"      //some/path/xyz/../to/some/venv//",
				"      stderr output",
			))
		})

		context("failure cases", func() {
			context("when executable returns an error", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = nil
					executable.ExecuteCall.Returns.Error = errors.New("could not run executable")
				})

				it("returns an error", func() {
					_, err := poetryInstallProcess.Execute(workingDir, packagesLayerPath, cacheLayerPath)
					Expect(err).To(MatchError("poetry install failed:\nerror: could not run executable"))
				})
			})
		})
	})
}
