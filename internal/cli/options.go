package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

var ErrHelp = flag.ErrHelp

type Config struct {
	LogDirectory  string
	ExecutionName string
	Quiet         bool
	Command       string
	Arguments     []string
}

func Parse(args []string, getenv func(string) string, output io.Writer) (Config, error) {
	var config Config
	flags := flag.NewFlagSet("citrace-shell", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&config.LogDirectory, "log-dir", "", "root directory for execution records")
	flags.StringVar(&config.ExecutionName, "execution-name", "", "human-readable execution name")
	flags.BoolVar(&config.Quiet, "quiet", false, "store output without mirroring it")
	flags.Usage = func() {
		fmt.Fprintln(output, "Usage: citrace-shell [options] -- <executable> [arguments...]")
		flags.PrintDefaults()
	}

	separator := -1
	for index, argument := range args {
		if argument == "--" {
			separator = index
			break
		}
	}
	if separator < 0 {
		return Config{}, errors.New("missing required -- separator")
	}
	if err := flags.Parse(args[:separator]); err != nil {
		return Config{}, err
	}
	command := args[separator+1:]
	if len(command) == 0 {
		return Config{}, errors.New("missing executable after --")
	}
	config.Command = command[0]
	config.Arguments = append([]string(nil), command[1:]...)

	if config.LogDirectory == "" {
		config.LogDirectory = ResolveLogDirectory(getenv)
	}
	if config.ExecutionName == "" {
		config.ExecutionName = getenv("CITRACE_EXECUTION_NAME")
	}
	quietWasSet := false
	flags.Visit(func(option *flag.Flag) {
		quietWasSet = quietWasSet || option.Name == "quiet"
	})
	if !quietWasSet {
		config.Quiet, _ = strconv.ParseBool(getenv("CITRACE_QUIET"))
	}
	return config, nil
}

func ResolveLogDirectory(getenv func(string) string) string {
	for _, candidate := range []struct {
		name   string
		suffix string
	}{
		{"CITRACE_LOG_DIR", ""},
		{"RUNNER_TEMP", "citrace"},
		{"AGENT_TEMPDIRECTORY", "citrace"},
		{"TMPDIR", "citrace"},
	} {
		if value := getenv(candidate.name); value != "" {
			if candidate.suffix != "" {
				return filepath.Join(value, candidate.suffix)
			}
			return value
		}
	}
	return filepath.Join(os.TempDir(), "citrace")
}
