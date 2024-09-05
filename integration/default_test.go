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

func testDefault(t *testing.T, context spec.G, it spec.S) {
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

			source, err = occam.Source(filepath.Join("testdata", "default_app"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds and runs successfully", func() {
			var err error
			var logs fmt.Stringer

			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.CPython.Online,
					settings.Buildpacks.Pip.Online,
					settings.Buildpacks.Poetry.Online,
					settings.Buildpacks.PoetryInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
			))
			Expect(logs).To(ContainLines("  Executing build process"))
			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(
					"    Running 'POETRY_CACHE_DIR=/layers/%s/cache POETRY_VIRTUALENVS_PATH=/layers/%s/poetry-venv poetry install --only main'",
					strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"),
					strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"),
				)),
			))
			Expect(logs).ToNot(ContainLines(
				MatchRegexp(`    \- Installing ipython \(\d+\.\d+\.\d+\)`),
			))
			Expect(logs).ToNot(ContainLines(
				MatchRegexp(`    \- Installing ruff \(\d+\.\d+\.\d+\)`),
			))
			Expect(logs).To(ContainLines(MatchRegexp(`      Completed in \d+\.\d+`)))
			Expect(logs).To(ContainLines(
				"  Configuring build environment",
				MatchRegexp(fmt.Sprintf(`    PATH                    -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/bin:\$PATH"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				MatchRegexp(fmt.Sprintf(`    POETRY_VIRTUALENVS_PATH -> "/layers/%s/poetry-venv"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				MatchRegexp(fmt.Sprintf(`    PYTHONPATH              -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/lib/python\d+\.\d+/site-packages:\$PYTHONPATH"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				"",
				"  Configuring launch environment",
				MatchRegexp(fmt.Sprintf(`    PATH                    -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/bin:\$PATH"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				MatchRegexp(fmt.Sprintf(`    POETRY_VIRTUALENVS_PATH -> "/layers/%s/poetry-venv"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				MatchRegexp(fmt.Sprintf(`    PYTHONPATH              -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/lib/python\d+\.\d+/site-packages:\$PYTHONPATH"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
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

		it("builds and runs successfully with develop dependencies", func() {
			var err error
			var logs fmt.Stringer

			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithEnv(map[string]string{
					"BP_POETRY_INSTALL_ONLY": "main,dev",
				}).
				WithBuildpacks(
					settings.Buildpacks.CPython.Online,
					settings.Buildpacks.Pip.Online,
					settings.Buildpacks.Poetry.Online,
					settings.Buildpacks.PoetryInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
			))
			Expect(logs).To(ContainLines("  Executing build process"))
			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(
					"    Running 'POETRY_CACHE_DIR=/layers/%s/cache POETRY_VIRTUALENVS_PATH=/layers/%s/poetry-venv poetry install --only main,dev'",
					strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"),
					strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"),
				)),
			))
			Expect(logs).To(ContainLines(
				MatchRegexp(`    \- Installing ipython \(\d+\.\d+\.\d+\)`),
			))
			Expect(logs).ToNot(ContainLines(
				MatchRegexp(`    \- Installing ruff \(\d+\.\d+\.\d+\)`),
			))
			Expect(logs).To(ContainLines(MatchRegexp(`      Completed in \d+\.\d+`)))
			Expect(logs).To(ContainLines(
				"  Configuring build environment",
				MatchRegexp(fmt.Sprintf(`    PATH                    -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/bin:\$PATH"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				MatchRegexp(fmt.Sprintf(`    POETRY_VIRTUALENVS_PATH -> "/layers/%s/poetry-venv"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				MatchRegexp(fmt.Sprintf(`    PYTHONPATH              -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/lib/python\d+\.\d+/site-packages:\$PYTHONPATH"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				"",
				"  Configuring launch environment",
				MatchRegexp(fmt.Sprintf(`    PATH                    -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/bin:\$PATH"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				MatchRegexp(fmt.Sprintf(`    POETRY_VIRTUALENVS_PATH -> "/layers/%s/poetry-venv"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
				MatchRegexp(fmt.Sprintf(`    PYTHONPATH              -> "/layers/%s/poetry-venv/default-app-.*-py\d+\.\d+/lib/python\d+\.\d+/site-packages:\$PYTHONPATH"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"))),
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
						"BP_LOG_LEVEL": "DEBUG",
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
					fmt.Sprintf("  Generating SBOM for /layers/%s/poetry-venv", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
					MatchRegexp(`      Completed in \d+(\.?\d+)*`),
				))
				Expect(logs).To(ContainLines(
					"  Writing SBOM in the following format(s):",
					"    application/vnd.cyclonedx+json",
					"    application/spdx+json",
					"    application/vnd.syft+json",
				))

				// check that all required SBOM files are present
				Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"), "poetry-venv", "sbom.cdx.json")).To(BeARegularFile())
				Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"), "poetry-venv", "sbom.spdx.json")).To(BeARegularFile())
				Expect(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"), "poetry-venv", "sbom.syft.json")).To(BeARegularFile())

				// check an SBOM file to make sure it has an entry for a poetry dependency
				contents, err := os.ReadFile(filepath.Join(sbomDir, "sbom", "launch", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_"), "poetry-venv", "sbom.cdx.json"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(`"name": "flask"`))
			})
		})
	})
}
