package poetryinstall_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	poetryinstall "github.com/paketo-buildpacks/poetry-install"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		detect     packit.DetectFunc
		workingDir string
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		err = os.WriteFile(filepath.Join(workingDir, "pyproject.toml"), []byte{}, 0644)
		Expect(err).NotTo(HaveOccurred())

		detect = poetryinstall.Detect()
	})

	context("detection", func() {
		it("returns a build plan that provides poetry virtual environment", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: poetryinstall.PoetryVenv},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: poetryinstall.CPython,
						Metadata: poetryinstall.BuildPlanMetadata{
							Build: true,
						},
					},
					{
						Name: poetryinstall.Poetry,
						Metadata: poetryinstall.BuildPlanMetadata{
							Build: true,
						},
					},
				},
			}))
		})

		context("when there is no pyproject.toml file", func() {
			it.Before(func() {
				Expect(os.Remove(filepath.Join(workingDir, "pyproject.toml"))).To(Succeed())
			})

			it("fails detection", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(packit.Fail.WithMessage("no 'pyproject.toml' found")))
			})
		})

		context("failure cases", func() {
			context("when the pyproject.toml file cannot be read", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := detect(packit.DetectContext{
						WorkingDir: workingDir,
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})

	})
}
