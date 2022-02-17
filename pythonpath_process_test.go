package poetryinstall_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	poetryinstall "github.com/paketo-buildpacks/poetry-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPythonPathProcess(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		venvDir string

		pythonPathProcess poetryinstall.PythonPathProcess
	)

	it.Before(func() {
		var err error
		venvDir, err = ioutil.TempDir("", "poetry-venv")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(venvDir, "lib", "python3.8", "site-packages"), os.ModePerm)).To(Succeed())

		pythonPathProcess = poetryinstall.NewPythonPathProcess()
	})

	it.After(func() {
		Expect(os.RemoveAll(venvDir)).To(Succeed())
	})

	context("Execute", func() {
		it("runs installation", func() {
			pythonPath, err := pythonPathProcess.Execute(venvDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(pythonPath).To(Equal(filepath.Join(venvDir, "lib", "python3.8", "site-packages")))
		})

		context("failure cases", func() {
			context("when the venvDir/lib directory cannot be read", func() {
				it.Before(func() {
					Expect(os.Chmod(filepath.Join(venvDir, "lib"), 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(filepath.Join(venvDir, "lib"), os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := pythonPathProcess.Execute(venvDir)
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to read directory: '%s'", filepath.Join(venvDir, "lib")))))
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when there are too many entries in the venv/lib directory", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(venvDir, "lib", "additional"), os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := pythonPathProcess.Execute(venvDir)
					Expect(err).To(MatchError(fmt.Sprintf("expected one directory and zero files in directory: '%s' - found multiple", filepath.Join(venvDir, "lib"))))
				})
			})

			context("If the entry at venv/lib/python* is not a directory", func() {
				it.Before(func() {
					Expect(os.RemoveAll(filepath.Join(venvDir, "lib", "python3.8"))).To(Succeed())

					_, err := os.Create(filepath.Join(venvDir, "lib", "python3.8"))
					Expect(err).NotTo(HaveOccurred())
				})

				it("returns an error", func() {
					_, err := pythonPathProcess.Execute(venvDir)
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("expected a directory at: '%s'", filepath.Join(venvDir, "lib", "python3.8")))))
				})
			})

			context("when the venvDir/lib/python* directory cannot be read", func() {
				it.Before(func() {
					Expect(os.Chmod(filepath.Join(venvDir, "lib", "python3.8"), 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(filepath.Join(venvDir, "lib", "python3.8"), os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := pythonPathProcess.Execute(venvDir)
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to read directory: '%s'", filepath.Join(venvDir, "lib", "python3.8")))))
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when there are too many entries in the venv/lib/python* directory", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(venvDir, "lib", "python3.8", "additional"), os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := pythonPathProcess.Execute(venvDir)
					Expect(err).To(MatchError(fmt.Sprintf("expected one directory and zero files in directory: '%s' - found multiple", filepath.Join(venvDir, "lib", "python3.8"))))
				})
			})

			context("If the only entry under venv/lib/python*/ is not called site-packages", func() {
				it.Before(func() {
					Expect(os.RemoveAll(filepath.Join(venvDir, "lib", "python3.8", "site-packages"))).To(Succeed())

					Expect(os.MkdirAll(filepath.Join(venvDir, "lib", "python3.8", "other-directory"), os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := pythonPathProcess.Execute(venvDir)
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf(`expected "site-packages" directory at: '%s'`, filepath.Join(venvDir, "lib", "python3.8", "site-packages")))))
				})
			})

			context("If the entry at venv/lib/python*/site-packages is not a directory", func() {
				it.Before(func() {
					Expect(os.RemoveAll(filepath.Join(venvDir, "lib", "python3.8", "site-packages"))).To(Succeed())

					_, err := os.Create(filepath.Join(venvDir, "lib", "python3.8", "site-packages"))
					Expect(err).NotTo(HaveOccurred())
				})

				it("returns an error", func() {
					_, err := pythonPathProcess.Execute(venvDir)
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("expected a directory at: '%s'", filepath.Join(venvDir, "lib", "python3.8", "site-packages")))))
				})
			})
		})
	})
}
