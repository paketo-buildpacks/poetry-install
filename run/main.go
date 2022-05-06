package main

import (
	"os"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	poetryinstall "github.com/paketo-buildpacks/poetry-install"
)

func main() {
	logger := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))

	packit.Run(
		poetryinstall.Detect(),
		poetryinstall.Build(
			draft.NewPlanner(),
			poetryinstall.NewPoetryInstallProcess(pexec.NewExecutable("poetry"), logger),
			poetryinstall.NewPythonPathProcess(),
			chronos.DefaultClock,
			logger,
		),
	)
}
