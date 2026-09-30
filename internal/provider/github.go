package provider

import "strings"

func github(getenv func(string) string) Provider {
	return environmentProvider{
		name:   "github-actions",
		getenv: getenv,
		detect: func() bool { return strings.EqualFold(getenv("GITHUB_ACTIONS"), "true") },
		allowlist: map[string]string{
			"GITHUB_RUN_ID":      "runId",
			"GITHUB_RUN_ATTEMPT": "runAttempt",
			"GITHUB_JOB":         "job",
			"GITHUB_WORKFLOW":    "workflow",
			"GITHUB_REPOSITORY":  "repository",
			"GITHUB_SHA":         "commit",
			"GITHUB_REF":         "ref",
			"GITHUB_ACTOR":       "actor",
			"GITHUB_WORKSPACE":   "workspace",
			"RUNNER_NAME":        "runnerName",
			"RUNNER_OS":          "runnerOS",
			"RUNNER_ARCH":        "runnerArch",
		},
	}
}
