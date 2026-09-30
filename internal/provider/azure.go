package provider

import "strings"

func azure(getenv func(string) string) Provider {
	return environmentProvider{
		name:   "azure-devops",
		getenv: getenv,
		detect: func() bool { return strings.EqualFold(getenv("TF_BUILD"), "true") },
		allowlist: map[string]string{
			"BUILD_BUILDID":           "buildId",
			"BUILD_BUILDNUMBER":       "buildNumber",
			"BUILD_DEFINITIONNAME":    "definitionName",
			"BUILD_REPOSITORY_NAME":   "repository",
			"BUILD_SOURCEVERSION":     "commit",
			"BUILD_SOURCEBRANCH":      "ref",
			"SYSTEM_JOBID":            "jobId",
			"SYSTEM_JOBDISPLAYNAME":   "job",
			"SYSTEM_STAGEID":          "stageId",
			"SYSTEM_STAGEDISPLAYNAME": "stage",
			"AGENT_NAME":              "agentName",
			"AGENT_OS":                "agentOS",
			"AGENT_TEMPDIRECTORY":     "agentTempDirectory",
			"BUILD_SOURCESDIRECTORY":  "sourcesDirectory",
		},
	}
}
