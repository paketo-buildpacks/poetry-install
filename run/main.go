package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/draft"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
	poetryinstall "github.com/paketo-buildpacks/poetry-install"
)

func main() {
	logger := scribe.NewEmitter(os.Stdout)

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
