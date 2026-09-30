## 1. Overview

CITrace Shell is a small, native Linux executable that acts as a transparent shell launcher for CI/CD pipeline steps.

The CI/CD runner invokes CITrace Shell instead of invoking Bash directly. CITrace Shell records execution metadata and logs, delegates execution to the configured real shell, forwards signals, and exits with the same result as the delegated shell.

Example:

```text
CI runner
  └── citrace-shell
        └── /bin/bash
              └── generated pipeline script
```

Example invocation:

```bash
citrace-shell -- /bin/bash --noprofile --norc -eo pipefail /tmp/pipeline-step.sh
```

The first version will not use `ptrace`, eBPF, kernel modules, or privileged container capabilities.

## 2. Goals

The minimum viable product must:

1. Run natively on Linux.
2. Execute a real shell or another binary.
3. Preserve command arguments without rewriting them.
4. Stream stdout and stderr to the CI/CD console.
5. Store stdout and stderr in local log files.
6. Record execution metadata in JSON.
7. Preserve the child process exit code.
8. Forward termination signals to the entire child process group.
9. Work inside an ordinary, unprivileged Linux container.
10. Support GitHub Actions through its custom-shell configuration.
11. Avoid exposing secrets through unnecessary environment capture.

## 3. Non-goals

The initial version will not:

- Trace system calls.
- Inspect file activity.
- Inspect network connections.
- Capture every process started by compiled applications.
- Replace `strace`.
- Require eBPF or `ptrace`.
- Provide Windows or macOS support.
- Upload logs to a remote service.
- Include a web interface.
- Modify pipeline scripts.
- Automatically redact arbitrary secrets from script contents.
- Transparently instrument third-party `uses:` actions.
- Globally replace `/bin/bash`.
- Provide an Azure DevOps extension in the first release.

These capabilities may be considered after the shell-launching foundation is stable.

## 4. Primary use cases

### 4.1 GitHub Actions custom shell

```yaml
defaults:
  run:
    shell: citrace-shell -- /bin/bash --noprofile --norc -eo pipefail {0}

steps:
  - name: Build
    run: |
      make build

  - name: Test
    run: |
      make test
```

Each `run` step produces its own execution directory.

### 4.2 Direct command execution

```bash
citrace-shell -- /bin/bash -c 'make build'
```

### 4.3 Script execution

```bash
citrace-shell -- /bin/bash -eo pipefail ./build.sh
```

### 4.4 Container execution

```bash
docker run --rm \
  -v "$PWD/logs:/logs" \
  -e CITRACE_LOG_DIR=/logs \
  citrace-shell \
  -- /bin/sh -c 'echo hello'
```

### 4.5 Azure DevOps initial integration

Azure DevOps does not offer the same general custom-shell interface for Bash steps. Initial Azure integration will therefore use explicit invocation:

```yaml
- bash: |
    citrace-shell -- /bin/bash -eo pipefail ./build.sh
  displayName: Traced build
```

A native Azure DevOps custom task can be implemented later.

## 5. Command-line interface

### 5.1 Basic syntax

```text
citrace-shell [citrace options] -- <executable> [arguments...]
```

Example:

```bash
citrace-shell \
  --log-dir /tmp/citrace \
  --execution-name build \
  -- /bin/bash -eo pipefail /tmp/build.sh
```

Everything after `--` must be passed to the child process without semantic modification.

### 5.2 Initial options

| Option | Environment variable | Description |
|---|---|---|
| `--log-dir` | `CITRACE_LOG_DIR` | Root directory for execution records |
| `--execution-name` | `CITRACE_EXECUTION_NAME` | Optional human-readable execution name |
| `--quiet` | `CITRACE_QUIET` | Store output without mirroring it to the console |
| `--version` | — | Print version information |
| `--help` | — | Print usage information |

Command-line options take precedence over environment variables.

### 5.3 Default log directory

The resolution order should be:

1. `--log-dir`
2. `CITRACE_LOG_DIR`
3. `$RUNNER_TEMP/citrace` on GitHub Actions
4. `$AGENT_TEMPDIRECTORY/citrace` on Azure DevOps
5. `$TMPDIR/citrace`
6. `/tmp/citrace`

The tool must not default to a directory that normally requires root access.

## 6. Output structure

Each invocation creates an isolated execution directory:

```text
<log-root>/
└── <execution-id>/
    ├── execution.json
    ├── stdout.log
    ├── stderr.log
    └── script.sh
```

`script.sh` is created only when the tool confidently identifies a readable script-file argument.

Example execution ID:

```text
20260930T182031.123456789Z-1042-a81f
```

The ID contains:

- UTC timestamp
- CITrace process ID
- Short random suffix

This prevents collisions when multiple steps start concurrently.

## 7. Execution record

The initial JSON schema should resemble:

```json
{
  "schemaVersion": "1.0",
  "id": "20260930T182031.123456789Z-1042-a81f",
  "name": "build",
  "provider": "github-actions",
  "executable": "/bin/bash",
  "arguments": [
    "--noprofile",
    "--norc",
    "-eo",
    "pipefail",
    "/tmp/pipeline-step.sh"
  ],
  "workingDirectory": "/workspace/project",
  "pid": 1048,
  "processGroupId": 1048,
  "startedAt": "2026-09-30T18:20:31.123456789Z",
  "finishedAt": "2026-09-30T18:20:44.456789123Z",
  "durationMs": 13333,
  "exitCode": 0,
  "signal": "",
  "stdoutLog": "stdout.log",
  "stderrLog": "stderr.log",
  "scriptSnapshot": "script.sh",
  "ci": {
    "repository": "company/service",
    "commit": "abc123",
    "runId": "123456789",
    "job": "build"
  }
}
```

### 7.1 Record lifecycle

The record must be written at least twice:

1. Immediately after starting the child, with `exitCode: null`.
2. After the child exits, with the final result.

Writes must use a temporary file followed by an atomic rename:

```text
execution.json.tmp → execution.json
```

This reduces the chance of leaving malformed JSON after interruption.

## 8. CI provider detection

Define a minimal provider interface:

```go
type Provider interface {
    Name() string
    Detect() bool
    Metadata() Metadata
}
```

Initial providers:

- GitHub Actions
- Azure DevOps
- Generic CI
- Local execution

### 8.1 GitHub Actions detection

Detect using:

```text
GITHUB_ACTIONS=true
```

Allowlist useful metadata:

- `GITHUB_RUN_ID`
- `GITHUB_RUN_ATTEMPT`
- `GITHUB_JOB`
- `GITHUB_WORKFLOW`
- `GITHUB_REPOSITORY`
- `GITHUB_SHA`
- `GITHUB_REF`
- `GITHUB_ACTOR`
- `GITHUB_WORKSPACE`
- `RUNNER_NAME`
- `RUNNER_OS`
- `RUNNER_ARCH`

### 8.2 Azure DevOps detection

Detect using:

```text
TF_BUILD=True
```

Allowlist useful metadata:

- `BUILD_BUILDID`
- `BUILD_BUILDNUMBER`
- `BUILD_DEFINITIONNAME`
- `BUILD_REPOSITORY_NAME`
- `BUILD_SOURCEVERSION`
- `BUILD_SOURCEBRANCH`
- `SYSTEM_JOBID`
- `SYSTEM_JOBDISPLAYNAME`
- `SYSTEM_STAGEID`
- `SYSTEM_STAGEDISPLAYNAME`
- `AGENT_NAME`
- `AGENT_OS`
- `AGENT_TEMPDIRECTORY`
- `BUILD_SOURCESDIRECTORY`

### 8.3 Environment policy

Do not save the complete process environment.

Only explicitly allowlisted, non-secret metadata should be included in `execution.json`. Variables containing terms such as the following must be excluded by default:

```text
TOKEN
SECRET
PASSWORD
PASSWD
KEY
CREDENTIAL
AUTH
COOKIE
SESSION
```

## 9. Process execution behavior

### 9.1 Executable resolution

The tool should resolve the child executable using `exec.LookPath` when a non-absolute executable is provided.

Both values may be recorded:

```json
{
  "requestedExecutable": "bash",
  "resolvedExecutable": "/usr/bin/bash"
}
```

### 9.2 Process group

The child must run in a separate process group:

```go
SysProcAttr: &syscall.SysProcAttr{
    Setpgid: true,
}
```

This allows signals to be forwarded to the complete process tree using:

```go
syscall.Kill(-processGroupID, signal)
```

### 9.3 Signal forwarding

Forward at least:

- `SIGINT`
- `SIGTERM`
- `SIGHUP`
- `SIGQUIT`

Expected behavior:

1. CITrace receives a signal.
2. CITrace sends the same signal to the child process group.
3. CITrace waits for the child to exit.
4. CITrace writes the final record where possible.
5. CITrace exits using the child’s resulting status.

A forced termination timeout may be added later but is not required for the MVP.

### 9.4 Exit codes

CITrace must return the child’s exit code.

For signal termination, use the conventional value:

```text
128 + signal number
```

Reserved launcher failures:

| Exit code | Meaning |
|---:|---|
| `125` | CITrace internal failure |
| `126` | Executable found but could not be started |
| `127` | Executable not found |

Where a real child exits with one of these values, CITrace must still preserve it.

## 10. Output capture

### 10.1 Standard behavior

stdout and stderr should remain separate:

```text
stdout → terminal and stdout.log
stderr → terminal and stderr.log
```

Implementation:

```go
cmd.Stdout = io.MultiWriter(os.Stdout, stdoutFile)
cmd.Stderr = io.MultiWriter(os.Stderr, stderrFile)
```

### 10.2 Backpressure

The initial implementation can use synchronous `io.MultiWriter`. This naturally applies backpressure instead of allowing unbounded in-memory buffering.

The tool must not collect complete output in memory.

### 10.3 Binary output

Log files should be treated as byte streams. CITrace must not assume that output is valid UTF-8.

### 10.4 Console control sequences

ANSI control sequences should remain unchanged in the raw logs for the MVP. Optional sanitized or plain-text rendering can be added later.

## 11. Script snapshot

When the final argument appears to be a readable regular file, CITrace may copy it into the execution directory.

Requirements:

- Use a maximum configured size.
- Copy before execution to capture the original content.
- Use restrictive permissions.
- Do not fail the execution if copying fails.
- Record snapshot failure as a warning in metadata.

Default maximum:

```text
1 MiB
```

Potential secret exposure must be documented. Script snapshots should be disableable in a later release if enabled by default.

## 12. Project structure

```text
citrace-shell/
├── cmd/
│   └── citrace-shell/
│       └── main.go
├── internal/
│   ├── app/
│   │   └── app.go
│   ├── cli/
│   │   └── options.go
│   ├── execution/
│   │   ├── command.go
│   │   ├── result.go
│   │   └── signals_linux.go
│   ├── logging/
│   │   ├── files.go
│   │   └── record.go
│   ├── provider/
│   │   ├── provider.go
│   │   ├── github.go
│   │   ├── azure.go
│   │   └── local.go
│   └── version/
│       └── version.go
├── testdata/
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
├── README.md
└── PLAN.md
```

Avoid introducing abstractions that are not needed by the MVP. The provider interface and execution/logging separation are the main boundaries.

## 13. Proposed Go types

```go
type Config struct {
    LogDirectory string
    ExecutionName string
    Quiet         bool
    Command       string
    Arguments     []string
}

type ExecutionRecord struct {
    SchemaVersion      string         `json:"schemaVersion"`
    ID                 string         `json:"id"`
    Name               string         `json:"name,omitempty"`
    Provider           string         `json:"provider"`
    RequestedExecutable string        `json:"requestedExecutable"`
    ResolvedExecutable string         `json:"resolvedExecutable"`
    Arguments          []string       `json:"arguments"`
    WorkingDirectory   string         `json:"workingDirectory"`
    PID                int            `json:"pid,omitempty"`
    ProcessGroupID     int            `json:"processGroupId,omitempty"`
    StartedAt          time.Time      `json:"startedAt"`
    FinishedAt         *time.Time     `json:"finishedAt,omitempty"`
    DurationMS         *int64         `json:"durationMs,omitempty"`
    ExitCode           *int           `json:"exitCode"`
    Signal             string         `json:"signal,omitempty"`
    Error              string         `json:"error,omitempty"`
    StdoutLog          string         `json:"stdoutLog"`
    StderrLog          string         `json:"stderrLog"`
    ScriptSnapshot     string         `json:"scriptSnapshot,omitempty"`
    CI                 map[string]string `json:"ci,omitempty"`
}
```

## 14. Security requirements

### 14.1 Do not capture the full environment

Environment values are a major source of CI/CD credentials. The report must use an allowlist rather than a denylist alone.

### 14.2 Command arguments

Arguments can contain credentials:

```bash
curl -H "Authorization: Bearer secret" ...
```

For the MVP:

- Preserve arguments in memory for child execution.
- Redact commonly recognizable secret flags before writing metadata.
- Do not modify arguments passed to the child.

Recognized forms should include:

```text
--token value
--token=value
--password value
--password=value
--api-key value
--authorization value
```

Recorded value:

```text
--token=[REDACTED]
```

### 14.3 File permissions

Recommended defaults:

```text
execution directory: 0750
log files:           0640
metadata:            0640
script snapshot:     0600
```

Support the caller’s `umask`.

### 14.4 Symbolic links

Prevent log path attacks:

- Create a new execution directory with a unique unpredictable suffix.
- Do not follow preexisting symlinks for output files.
- Prefer `O_EXCL` when creating files.
- Reject a log root that is not a directory.
- Do not accept user-controlled filenames for individual log files.

### 14.5 Logging failures

The default policy should be fail-closed before execution:

- If the execution directory cannot be created, do not run the child.
- If stdout/stderr logs cannot be created, do not run the child.
- If the final metadata write fails after execution, preserve the child exit code but emit a clear error to stderr.

A future `--best-effort` option could relax this policy.

## 15. Container support

The MVP must work without:

- `--privileged`
- `CAP_SYS_ADMIN`
- `CAP_BPF`
- Host PID namespace
- Mounted `/sys/kernel/debug`
- Mounted `/sys/fs/bpf`

Example image:

```dockerfile
FROM golang:1.25 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/citrace-shell \
    ./cmd/citrace-shell

FROM scratch
COPY --from=build /out/citrace-shell /citrace-shell
ENTRYPOINT ["/citrace-shell"]
```

A `scratch` image can launch binaries available through mounted paths, but it will not contain Bash itself. For general interactive testing, an Alpine or Debian runtime image may be more convenient.

When used as a shell inside an existing CI job container, install or mount the static CITrace binary into that container.

## 16. Testing strategy

### 16.1 Unit tests

Test:

- CLI parsing around `--`
- Log-directory resolution
- CI provider detection
- Metadata allowlisting
- Argument redaction
- Exit-code conversion
- Signal-to-exit-code conversion
- Execution ID uniqueness
- Atomic JSON writing
- Script-file detection

### 16.2 Integration tests

Execute small fixture programs that:

1. Exit successfully.
2. Exit with a nonzero code.
3. Write separately to stdout and stderr.
4. Produce large output.
5. Spawn child processes.
6. Handle `SIGTERM`.
7. Ignore `SIGTERM`.
8. Terminate from a signal.
9. Write binary/non-UTF-8 output.
10. Use arguments containing spaces and special characters.

Example fixture:

```go
package main

import (
    "fmt"
    "os"
)

func main() {
    fmt.Fprintln(os.Stdout, "stdout message")
    fmt.Fprintln(os.Stderr, "stderr message")
    os.Exit(17)
}
```

Assertions:

- Console output is preserved.
- Log files contain the correct bytes.
- `execution.json` contains exit code `17`.
- CITrace exits with code `17`.

### 16.3 Signal tests

Start a fixture that creates child processes, then signal CITrace:

```text
test process
  └── citrace-shell
        └── bash
              └── sleep
```

Verify that the signal reaches both Bash and its descendants and that no orphaned processes remain.

### 16.4 Container tests

Run the integration suite inside:

- Debian
- Ubuntu
- Alpine
- A non-root container
- GitHub Actions job container

### 16.5 Race and quality checks

CI should run:

```bash
go test ./...
go test -race ./...
go vet ./...
staticcheck ./...
```

## 17. Build and release

### 17.1 Supported architecture for MVP

Start with:

```text
linux/amd64
```

Add after initial validation:

```text
linux/arm64
```

### 17.2 Build command

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build \
  -trimpath \
  -ldflags="-s -w -X main.version=${VERSION}" \
  -o dist/citrace-shell-linux-amd64 \
  ./cmd/citrace-shell
```

### 17.3 Release artifacts

Each release should include:

- Static Linux AMD64 binary
- Static Linux ARM64 binary
- SHA-256 checksums
- Container image
- Software bill of materials
- Signed release artifacts, when signing infrastructure is available

## 18. Delivery milestones

### Milestone 1: Process launcher

Deliver:

- CLI parser
- Command execution
- stdout/stderr streaming
- Separate log files
- Exit-code preservation
- Basic execution record

Acceptance criteria:

```bash
citrace-shell -- /bin/sh -c 'echo out; echo err >&2; exit 7'
```

must:

- Print both messages.
- Write both log files.
- Record exit code `7`.
- Exit with code `7`.

### Milestone 2: Linux process lifecycle

Deliver:

- Separate process group
- Signal forwarding
- Signal result recording
- Atomic metadata updates
- Initial metadata written before waiting

Acceptance criteria:

- Sending `SIGTERM` to CITrace also terminates the child process tree.
- The record identifies signal-based termination.
- No child process remains after normal signal handling.

### Milestone 3: CI awareness

Deliver:

- GitHub Actions provider
- Azure DevOps provider
- Generic/local provider
- Safe metadata allowlist
- Automatic log-root selection

Acceptance criteria:

- Reports identify the provider.
- No unapproved environment variables appear in the report.
- GitHub run/job metadata is associated with the execution.

### Milestone 4: Security hardening

Deliver:

- Argument redaction
- Safe file creation
- Restrictive permissions
- Script snapshot size limit
- Security documentation

Acceptance criteria:

- Known secret argument patterns are redacted.
- Logs cannot overwrite arbitrary files through symlinks.
- The child is not executed if required logs cannot be opened.

### Milestone 5: Distribution

Deliver:

- Static AMD64 and ARM64 binaries
- Container image
- GitHub Actions usage documentation
- Azure DevOps explicit-invocation documentation
- Checksums and release process

## 19. Future work

After the MVP is stable, possible extensions include:

1. Azure DevOps custom task.
2. Reusable GitHub Action for installation.
3. Combined chronological stdout/stderr event stream.
4. Optional Bash xtrace capture.
5. Log compression.
6. Configurable retention.
7. Remote object-storage upload.
8. OpenTelemetry export.
9. SARIF policy findings.
10. Artifact hashing.
11. Process-tree observation through `/proc`.
12. Optional `ptrace` backend.
13. Optional eBPF backend.
14. File and network access tracing.
15. Step-to-step execution comparison.
16. Cryptographically signed execution records.

## 20. Key design decisions

### Decision: shell launcher rather than kernel tracer

The MVP prioritizes portability, safety, and deployment simplicity. It will run in an unprivileged container and on ordinary Linux CI runners.

### Decision: preserve the real shell

CITrace is not a shell-language interpreter. It delegates all script interpretation to Bash, `sh`, or another caller-selected executable.

### Decision: explicit `--` separator

The separator prevents CITrace options from conflicting with delegated shell options.

### Decision: separate stdout and stderr

Separate files preserve stream identity and are more useful for troubleshooting. Both streams remain visible in the normal CI console.

### Decision: no complete environment capture

The security risk of leaking credentials outweighs the diagnostic value. Only allowlisted CI metadata is recorded.

### Decision: preserve child status

CITrace instrumentation must not turn a failing build into a successful build or vice versa.

## 21. Definition of done

The MVP is complete when:

- A static Go binary runs on Linux AMD64.
- It can be selected as a GitHub Actions custom shell.
- It can explicitly launch Bash in Azure DevOps.
- It passes arbitrary shell arguments without modification.
- It stores stdout, stderr, and structured metadata.
- It returns the delegated process’s exit code.
- It forwards common termination signals to the child process group.
- It runs as a non-root user in an ordinary container.
- It does not capture the complete environment.
- Integration tests cover success, failure, signals, output, and containers.
- Installation and usage are documented.