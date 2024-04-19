package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testWithDependencyGroup(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when the buildpack is run with pack build", func() {
		var (
			image     occam.Image
			container occam.Container
			name      string
			source    string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			source, err = occam.Source(filepath.Join("testdata", "app_with_dependency_group"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds and runs successfully with dev dependency group", func() {
			var err error
			var logs fmt.Stringer
			buildPackIdUnderscore := strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")

			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.CPython.Online,
					settings.Buildpacks.Pip.Online,
					settings.Buildpacks.Poetry.Online,
					settings.Buildpacks.PoetryInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				WithEnv(map[string]string{"BP_POETRY_INSTALL_WITH": "dev"}).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
				"  Executing build process",
				MatchRegexp(fmt.Sprintf(
					"    Running 'POETRY_CACHE_DIR=/layers/%s/cache POETRY_VIRTUALENVS_PATH=/layers/%s/poetry-venv poetry install --with dev'",
					buildPackIdUnderscore,
					buildPackIdUnderscore,
				)),
			))

			aDependencyFromMainGroup := "Installing flask"
			aDependencyFromDevGroup := "Installing pytest"
			Expect(logs).To(ContainSubstring(aDependencyFromMainGroup))
			Expect(logs).To(ContainSubstring(aDependencyFromDevGroup))

			Expect(logs).To(ContainLines(MatchRegexp(`      Completed in \d+\.\d+`)))

			Expect(logs).To(ContainLines(
				"  Configuring build environment",
				MatchRegexp(fmt.Sprintf(`    PATH                    -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/bin:\$PATH"`, buildPackIdUnderscore)),
				MatchRegexp(fmt.Sprintf(`    POETRY_VIRTUALENVS_PATH -> "/layers/%s/poetry-venv"`, buildPackIdUnderscore)),
				MatchRegexp(fmt.Sprintf(`    PYTHONPATH              -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/lib/python\d+\.\d+/site-packages:\$PYTHONPATH"`, buildPackIdUnderscore)),
			))
			Expect(logs).To(ContainLines(
				"  Configuring launch environment",
				MatchRegexp(fmt.Sprintf(`    PATH                    -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/bin:\$PATH"`, buildPackIdUnderscore)),
				MatchRegexp(fmt.Sprintf(`    POETRY_VIRTUALENVS_PATH -> "/layers/%s/poetry-venv"`, buildPackIdUnderscore)),
				MatchRegexp(fmt.Sprintf(`    PYTHONPATH              -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/lib/python\d+\.\d+/site-packages:\$PYTHONPATH"`, buildPackIdUnderscore)),
			))

			container, err = docker.Container.Run.
				WithCommand("gunicorn server:app").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(image.ID)
			Expect(err).ToNot(HaveOccurred())

			Eventually(container).Should(BeAvailable())
			Eventually(container).Should(Serve(ContainSubstring("Hello, World!")).OnPort(8080))
		})

		context("validating SBOM", func() {
			var (
				sbomDir string
			)

			it.Before(func() {
				var err error
				sbomDir, err = os.MkdirTemp("", "sbom")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.Chmod(sbomDir, os.ModePerm)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(sbomDir)).To(Succeed())
			})

			it("writes SBOM files to the layer and label metadata", func() {
				var err error
				var logs fmt.Stringer
				buildPackIdUnderscore := strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")

				image, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						settings.Buildpacks.CPython.Online,
						settings.Buildpacks.Pip.Online,
						settings.Buildpacks.Poetry.Online,
						settings.Buildpacks.PoetryInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					WithEnv(map[string]string{
						"BP_LOG_LEVEL":           "DEBUG",
						"BP_POETRY_INSTALL_WITH": "dev",
					}).
					WithSBOMOutputDir(sbomDir).
					Execute(name, source)
				Expect(err).ToNot(HaveOccurred(), logs.String)

				container, err = docker.Container.Run.
					WithCommand("gunicorn server:app").
					WithEnv(map[string]string{"PORT": "8080"}).
					WithPublish("8080").
					Execute(image.ID)
				Expect(err).ToNot(HaveOccurred())

				Eventually(container).Should(BeAvailable())
				Eventually(container).Should(Serve(ContainSubstring("Hello, World!")).OnPort(8080))

				Expect(logs).To(ContainLines(
					fmt.Sprintf("  Generating SBOM for /layers/%s/poetry-venv", buildPackIdUnderscore),
					MatchRegexp(`      Completed in \d+(\.?\d+)*`),
				))
				Expect(logs).To(ContainLines(
					"  Writing SBOM in the following format(s):",
					"    application/vnd.cyclonedx+json",
					"    application/spdx+json",
					"    application/vnd.syft+json",
				))

				// check that all required SBOM files are present
				Expect(filepath.Join(sbomDir, "sbom", "launch", buildPackIdUnderscore, "poetry-venv", "sbom.cdx.json")).To(BeARegularFile())
				Expect(filepath.Join(sbomDir, "sbom", "launch", buildPackIdUnderscore, "poetry-venv", "sbom.spdx.json")).To(BeARegularFile())
				Expect(filepath.Join(sbomDir, "sbom", "launch", buildPackIdUnderscore, "poetry-venv", "sbom.syft.json")).To(BeARegularFile())

				// check an SBOM file to make sure it has an entry for a poetry dependency
				contents, err := os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", buildPackIdUnderscore, "poetry-venv", "sbom.cdx.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(`"name": "flask"`))
				// optional dev dependencies should be included
				Expect(string(contents)).To(ContainSubstring(`"name": "pytest"`))
			})
		})
	})
}
