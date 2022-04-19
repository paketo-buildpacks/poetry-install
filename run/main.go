package main

import (
	"os"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	poetryinstall "github.com/paketo-buildpacks/poetry-install"
)

type Generator struct{}

func (f Generator) Generate(dir string) (sbom.SBOM, error) {
	return sbom.Generate(dir)
}

func main() {
	logger := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))

	packit.Run(
		poetryinstall.Detect(),
		poetryinstall.Build(
			draft.NewPlanner(),
			poetryinstall.NewPoetryInstallProcess(pexec.NewExecutable("poetry"), logger),
			poetryinstall.NewPythonPathProcess(),
			Generator{},
			chronos.DefaultClock,
			logger,
		),
	)
}
