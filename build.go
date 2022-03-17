package poetryinstall

import (
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
//go:generate faux --interface InstallProcess --output fakes/install_process.go
//go:generate faux --interface PythonPathLookupProcess --output fakes/python_path_process.go

// EntryResolver defines the interface for picking the most relevant entry from
// the Buildpack Plan entries.
type EntryResolver interface {
	MergeLayerTypes(name string, entries []packit.BuildpackPlanEntry) (launch, build bool)
}

// InstallProcess defines the interface for installing the poetry dependencies.
// It returns the location of the virtual env directory.
type InstallProcess interface {
	Execute(workingDir, targetDir, cacheDir string) (string, error)
}

// PythonPathProcess defines the interface for finding the PYTHONPATH (AKA the site-packages directory)
type PythonPathLookupProcess interface {
	Execute(venvDir string) (string, error)
}

// Build will return a packit.BuildFunc that will be invoked during the build
// phase of the buildpack lifecycle.
//
// Build will install the poetry dependencies by using the pyproject.toml file
// to a virtual environment layer.
func Build(entryResolver EntryResolver, installProcess InstallProcess, pythonPathProcess PythonPathLookupProcess, clock chronos.Clock, logger scribe.Emitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		venvLayer, err := context.Layers.Get(VenvLayerName)
		if err != nil {
			return packit.BuildResult{}, err
		}

		cacheLayer, err := context.Layers.Get(CacheLayerName)
		if err != nil {
			return packit.BuildResult{}, err
		}

		var venvDir string
		logger.Process("Executing build process")
		duration, err := clock.Measure(func() error {
			venvDir, err = installProcess.Execute(context.WorkingDir, venvLayer.Path, cacheLayer.Path)
			return err
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		venvLayer.Metadata = map[string]interface{}{
			"built_at": clock.Now().Format(time.RFC3339Nano),
		}

		pythonPathDir, err := pythonPathProcess.Execute(venvDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		venvLayer.Launch, venvLayer.Build = entryResolver.MergeLayerTypes(PoetryVenv, context.Plan.Entries)
		venvLayer.Cache = venvLayer.Launch || venvLayer.Build
		cacheLayer.Cache = true

		logger.Process("Configuring environment")
		venvLayer.SharedEnv.Default("POETRY_VIRTUALENVS_PATH", venvLayer.Path)
		venvLayer.SharedEnv.Prepend("PYTHONPATH", pythonPathDir, string(os.PathListSeparator))
		venvLayer.SharedEnv.Prepend("PATH", filepath.Join(venvDir, "bin"), string(os.PathListSeparator))
		logger.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(venvLayer.SharedEnv))
		logger.Break()

		layers := []packit.Layer{venvLayer}
		if _, err := os.Stat(cacheLayer.Path); err == nil {
			if !fs.IsEmptyDir(cacheLayer.Path) {
				layers = append(layers, cacheLayer)
			}
		}

		result := packit.BuildResult{
			Layers: layers,
		}

		return result, nil
	}
}
