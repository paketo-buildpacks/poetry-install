package poetryinstall_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	poetryinstall "github.com/paketo-buildpacks/poetry-install"
	"github.com/paketo-buildpacks/poetry-install/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string

		entryResolver     *fakes.EntryResolver
		installProcess    *fakes.InstallProcess
		pythonPathProcess *fakes.PythonPathLookupProcess

		buffer *bytes.Buffer

		build        packit.BuildFunc
		buildContext packit.BuildContext
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		installProcess = &fakes.InstallProcess{}
		installProcess.ExecuteCall.Returns.String = "some-venv-dir"

		pythonPathProcess = &fakes.PythonPathLookupProcess{}
		pythonPathProcess.ExecuteCall.Returns.String = "some-python-path"

		entryResolver = &fakes.EntryResolver{}

		buffer = bytes.NewBuffer(nil)

		build = poetryinstall.Build(
			entryResolver,
			installProcess,
			pythonPathProcess,
			chronos.DefaultClock,
			scribe.NewEmitter(buffer),
		)

		buildContext = packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				Version:     "some-version",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
			},
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{Name: "poetry-venv"},
				},
			},
			Platform: packit.Platform{Path: "some-platform-path"},
			Layers:   packit.Layers{Path: layersDir},
			Stack:    "some-stack",
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	it("runs the build process and returns expected layers", func() {
		result, err := build(buildContext)
		Expect(err).NotTo(HaveOccurred())

		layers := result.Layers
		Expect(layers).To(HaveLen(1))

		venvLayer := layers[0]
		Expect(venvLayer.Name).To(Equal("poetry-venv"))
		Expect(venvLayer.Path).To(Equal(filepath.Join(layersDir, "poetry-venv")))

		Expect(venvLayer.Build).To(BeFalse())
		Expect(venvLayer.Launch).To(BeFalse())
		Expect(venvLayer.Cache).To(BeFalse())

		Expect(venvLayer.BuildEnv).To(BeEmpty())
		Expect(venvLayer.LaunchEnv).To(BeEmpty())
		Expect(venvLayer.ProcessLaunchEnv).To(BeEmpty())

		Expect(venvLayer.SharedEnv).To(HaveLen(5))
		Expect(venvLayer.SharedEnv["PATH.prepend"]).To(Equal("some-venv-dir/bin"))
		Expect(venvLayer.SharedEnv["PATH.delim"]).To(Equal(":"))
		Expect(venvLayer.SharedEnv["PYTHONPATH.prepend"]).To(Equal("some-python-path"))
		Expect(venvLayer.SharedEnv["PYTHONPATH.delim"]).To(Equal(":"))
		Expect(venvLayer.SharedEnv["POETRY_VIRTUALENVS_PATH.default"]).To(Equal(filepath.Join(layersDir, "poetry-venv")))

		expectedSbom, err := sbom.Generate(workingDir)
		Expect(err).NotTo(HaveOccurred())

		Expect(venvLayer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
			{
				Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(expectedSbom, sbom.CycloneDXFormat),
			},
			{
				Extension: sbom.Format(sbom.SPDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(expectedSbom, sbom.SPDXFormat),
			},
		}))

		Expect(installProcess.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(installProcess.ExecuteCall.Receives.TargetDir).To(Equal(filepath.Join(layersDir, "poetry-venv")))
		Expect(installProcess.ExecuteCall.Receives.CacheDir).To(Equal(filepath.Join(layersDir, "cache")))

		Expect(pythonPathProcess.ExecuteCall.Receives.VenvDir).To(Equal("some-venv-dir"))

		Expect(entryResolver.MergeLayerTypesCall.Receives.Name).To(Equal("poetry-venv"))
		Expect(entryResolver.MergeLayerTypesCall.Receives.Entries).To(Equal([]packit.BuildpackPlanEntry{
			{Name: "poetry-venv"},
		}))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
	})

	context("poetry-venv is required at build and launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("layer's build, launch, cache flags must be set", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			layers := result.Layers
			Expect(layers).To(HaveLen(1))

			venvLayer := layers[0]
			Expect(venvLayer.Name).To(Equal("poetry-venv"))

			Expect(venvLayer.Build).To(BeTrue())
			Expect(venvLayer.Launch).To(BeTrue())
			Expect(venvLayer.Cache).To(BeTrue())
		})
	})

	context("poetry-venv is required only at launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = false
		})

		it("layer's build, cache flags must be set", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			layers := result.Layers
			Expect(layers).To(HaveLen(1))

			venvLayer := layers[0]
			Expect(venvLayer.Name).To(Equal("poetry-venv"))

			Expect(venvLayer.Build).To(BeFalse())
			Expect(venvLayer.Launch).To(BeTrue())
			Expect(venvLayer.Cache).To(BeTrue())

			Expect(venvLayer.BuildEnv).To(BeEmpty())
			Expect(venvLayer.LaunchEnv).To(BeEmpty())
			Expect(venvLayer.ProcessLaunchEnv).To(BeEmpty())

			Expect(venvLayer.SharedEnv).To(HaveLen(5))
			Expect(venvLayer.SharedEnv["PATH.prepend"]).To(Equal("some-venv-dir/bin"))
			Expect(venvLayer.SharedEnv["PATH.delim"]).To(Equal(":"))
			Expect(venvLayer.SharedEnv["PYTHONPATH.prepend"]).To(Equal("some-python-path"))
			Expect(venvLayer.SharedEnv["PYTHONPATH.delim"]).To(Equal(":"))
			Expect(venvLayer.SharedEnv["POETRY_VIRTUALENVS_PATH.default"]).To(Equal(filepath.Join(layersDir, "poetry-venv")))
		})
	})

	context("install process utilizes cache", func() {
		it.Before(func() {
			installProcess.ExecuteCall.Stub = func(_, _, cachePath string) (string, error) {
				err := os.MkdirAll(filepath.Join(cachePath, "something"), os.ModePerm)
				if err != nil {
					return "", fmt.Errorf("issue with stub call: %+v", err)
				}

				return "some-cached-venv-dir", nil
			}
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("result should include a cache layer", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			layers := result.Layers
			Expect(layers).To(HaveLen(2))

			venvLayer := layers[0]
			Expect(venvLayer.Name).To(Equal("poetry-venv"))

			Expect(venvLayer.Build).To(BeTrue())
			Expect(venvLayer.Launch).To(BeTrue())
			Expect(venvLayer.Cache).To(BeTrue())

			Expect(venvLayer.BuildEnv).To(BeEmpty())
			Expect(venvLayer.LaunchEnv).To(BeEmpty())
			Expect(venvLayer.ProcessLaunchEnv).To(BeEmpty())

			Expect(venvLayer.SharedEnv).To(HaveLen(5))
			Expect(venvLayer.SharedEnv["PATH.prepend"]).To(Equal("some-cached-venv-dir/bin"))
			Expect(venvLayer.SharedEnv["PATH.delim"]).To(Equal(":"))
			Expect(venvLayer.SharedEnv["PYTHONPATH.prepend"]).To(Equal("some-python-path"))
			Expect(venvLayer.SharedEnv["PYTHONPATH.delim"]).To(Equal(":"))
			Expect(venvLayer.SharedEnv["POETRY_VIRTUALENVS_PATH.default"]).To(Equal(filepath.Join(layersDir, "poetry-venv")))

			cacheLayer := layers[1]
			Expect(cacheLayer.Name).To(Equal("cache"))
			Expect(cacheLayer.Path).To(Equal(filepath.Join(layersDir, "cache")))
			Expect(cacheLayer.Build).To(BeFalse())
			Expect(cacheLayer.Launch).To(BeFalse())
			Expect(cacheLayer.Cache).To(BeTrue())
			Expect(cacheLayer.BuildEnv).To(BeEmpty())
			Expect(cacheLayer.LaunchEnv).To(BeEmpty())
			Expect(cacheLayer.ProcessLaunchEnv).To(BeEmpty())
			Expect(cacheLayer.SharedEnv).To(BeEmpty())
			Expect(cacheLayer.Metadata).To(BeEmpty())
		})
	})

	context("failure cases", func() {
		context("when the layers directory cannot be written to", func() {
			it.Before(func() {
				Expect(os.Chmod(layersDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when install process returns an error", func() {
			it.Before(func() {
				installProcess.ExecuteCall.Returns.Error = errors.New("could not run install process")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("could not run install process"))
			})
		})

		context("when Python path lookup process returns an error", func() {
			it.Before(func() {
				pythonPathProcess.ExecuteCall.Returns.Error = errors.New("could not run Python path process")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("could not run Python path process"))
			})
		})

		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				buildContext.BuildpackInfo.SBOMFormats = []string{"random-format"}
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(`unsupported SBOM format: 'random-format'`))
			})
		})
	})
}
