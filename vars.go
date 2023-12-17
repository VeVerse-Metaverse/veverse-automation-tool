package main

var (
	api2Url              string // APIv2 base URL
	wailsPath            string // Path to Wails CLI
	signToolPath         string // Path to SignTool
	certFile             string // Path to certificate file
	certPassword         string // Password for the certificate
	uatPath              string // Path to UAT
	uvsPath              string // Path to UVS
	editorPath           string // Path to UnrealEditor-Cmd
	projectDir           string // Path to the project
	launcherDir          string // Path to the launcher
	projectName          string // Project name
	apiEmail             string // Email of a builder to authenticate with APIv2
	apiPassword          string // Password of a builder to authenticate with APIv2
	platforms            string // Platforms supported by this automation tool
	jobTypes             string // Job types supported by this automation tool
	deployments          string // Deployments supported by this automation tool
	token                string // API v2 JWT
	ueVersionCode        string // Unreal Engine source code version
	ueVersionMarketplace string // Unreal Engine marketplace version
	supportedPlatforms   = map[string]bool{}
	supportedJobTypes    = map[string]bool{}
	supportedDeployments = map[string]bool{}
)

const (
	JobStatusUnclaimed = iota
	JobStatusClaimed
	JobStatusProcessing
	JobStatusUploading
	JobStatusCompleted
	JobStatusError
	JobStatusCancelled
)

var supportedJobStatuses = map[int]string{
	JobStatusUnclaimed:  "unclaimed",
	JobStatusClaimed:    "claimed",
	JobStatusProcessing: "processing",
	JobStatusUploading:  "uploading",
	JobStatusCompleted:  "completed",
	JobStatusError:      "error",
	JobStatusCancelled:  "cancelled",
}

var configurationBranchMapping = map[string]string{
	"Debug":       "development",
	"DebugGame":   "development",
	"Development": "development",
	"Test":        "test",
	"Shipping":    "release",
}
