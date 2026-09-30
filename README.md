# CITrace Shell

CITrace Shell is a native Linux launcher for CI/CD steps. It delegates execution to a real shell or binary, mirrors stdout and stderr to the job console, stores each stream separately, records execution metadata, forwards termination signals to the child process group, and returns the child's status.

## Build

Go 1.22 or newer is required.

```bash
make test
make build VERSION=0.1.0
./dist/citrace-shell --version
```

Static AMD64 and ARM64 release binaries and checksums are produced with:

```bash
make release VERSION=0.1.0
```

## Usage

Everything after the required `--` separator is passed to the child without rewriting:

```bash
citrace-shell -- /bin/sh -c 'echo out; echo err >&2; exit 7'
citrace-shell --log-dir /tmp/citrace --execution-name build -- /bin/bash -eo pipefail ./build.sh
```

Supported options are `--log-dir`, `--execution-name`, `--quiet`, `--help`, and `--version`. Their environment equivalents are `CITRACE_LOG_DIR`, `CITRACE_EXECUTION_NAME`, and `CITRACE_QUIET`. Command-line values take precedence.

Each invocation creates a unique directory containing `execution.json`, `stdout.log`, and `stderr.log`. If the final argument is a readable regular file no larger than 1 MiB, it is copied before execution as `script.sh`.

### GitHub Actions

Install `citrace-shell` on `PATH`, then configure it as the shell for `run` steps:

```yaml
defaults:
  run:
    shell: citrace-shell -- /bin/bash --noprofile --norc -eo pipefail {0}

steps:
  - run: make build
  - run: make test
```

The default log root is `$RUNNER_TEMP/citrace`.

### Azure DevOps

Invoke the launcher explicitly:

```yaml
- bash: |
    citrace-shell -- /bin/bash -eo pipefail ./build.sh
  displayName: Traced build
```

The default log root is `$AGENT_TEMPDIRECTORY/citrace`.

### Container

The minimal image runs as a numeric non-root user and includes BusyBox as `/bin/sh`:

```bash
docker build -t citrace-shell .
docker run --rm -v "$PWD/logs:/logs" -e CITRACE_LOG_DIR=/logs citrace-shell -- /bin/sh -c 'echo hello'
```

Ensure the mounted log directory is writable by the container user.

## Security

CITrace never records the complete environment. GitHub Actions and Azure DevOps metadata use explicit allowlists; generic CI and local runs include no environment metadata.

Arguments following `--token`, `--password`, `--api-key`, and `--authorization`, including `--name=value` forms, are redacted in `execution.json`. Arguments are not changed for child execution. Other credential formats may still appear in metadata or raw output.

Script snapshots can contain credentials embedded in CI scripts. Keep the log root access-controlled and retain or publish snapshots only when appropriate. Execution directories use mode `0750`, logs and metadata use `0640`, and snapshots use `0600`, subject to the caller's umask.

## Exit Status

Normal child exit codes are preserved. Signal termination returns `128 + signal number`. Launcher failures use `125` for internal failures, `126` when an executable cannot start, and `127` when it cannot be found.