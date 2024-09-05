# Poetry Install Cloud Native Buildpack
## `gcr.io/paketo-buildpacks/poetry-install`

The Paketo Buildpack for Poetry Install is a Cloud Native Buildpack that installs
packages using [Poetry](https://python-poetry.org/) and makes the installed packages
available to the application.

The buildpack is published for consumption at
`gcr.io/paketo-buildpacks/poetry-install` and `paketobuildpacks/poetry-install`.

## Behavior
This buildpack participates if `pyproject.toml` exists at the root the app.

The buildpack will do the following:
* At build time:
  - Creates a virtual environment, installs the application packages to it,
    and makes this virtual environment available to the app via a layer called `poetry-venv`.
  - Configures `poetry` to locate this virtual environment via the
    environment variable `POETRY_VIRTUAL_ENVS_PATH`.
  - Prepends the layer `poetry-venv` onto `PYTHONPATH`.
  - Prepends the `bin` directory of the `poetry-venv` layer to the `PATH` environment variable.
* At run time:
  - Does nothing

## Configuration
| Environment Variable | Description                                                                                                                                                                          |
|----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `$BP_POETRY_INSTALL_ONLY` | Configure which groups from `pyproject.toml` file will be installed, default is `main`. |

## Integration

The Poetry Install CNB provides `poetry-venv` as a dependency. Downstream
buildpacks can require the `poetry-venv` dependency by generating a [Build
Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the dependency provided by the Poetry Install Buildpack is
  # "poetry-venv". This value is considered part of the public API for the
  # buildpack and will not change without a plan for deprecation.
  name = "poetry-venv"

  # The Poetry Install buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the poetry-venv
    # dependency is available on the $PYTHONPATH for subsequent
    # buildpacks during their build phase. If you are writing a buildpack that
    # needs poetry-venv during its build process, this flag should be
    # set to true.
    build = true

    # Setting the launch flag to true will ensure that the poetry-venv
    # dependency is available on the $PYTHONPATH for the running
    # application. If you are writing an application that needs poetry-venv
    # at runtime, this flag should be set to true.
    launch = true
```

## Usage

To package this buildpack for consumption:
```
$ ./scripts/package.sh --version x.x.x
```
This will create a `buildpackage.cnb` file under the build directory which you
can use to build your app as follows: `pack build <app-name> -p <path-to-app>
-b <cpython buildpack> -b <poetry buildpack> -b build/buildpackage.cnb -b
<other-buildpacks..>`.

To run the unit and integration tests for this buildpack:
```
$ ./scripts/unit.sh && ./scripts/integration.sh
```

## Known issues and limitations

* This buildpack will not work in an offline/air-gapped environment: vendoring
  of dependencies is not supported. This is a limitation of `poetry` - which
  itself does not support vendoring dependencies.
