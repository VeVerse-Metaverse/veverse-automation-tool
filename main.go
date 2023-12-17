package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var rootCmd *cobra.Command

func init() {
	api2Url = os.Getenv("VAT_API2_URL")
	if api2Url == "" {
		Logger.Fatalln("required env VAT_API2_URL is not defined")
	}

	projectDir = os.Getenv("VAT_PROJECT_DIR")
	if projectDir == "" {
		Logger.Fatalln("required env VAT_PROJECT_DIR is not defined")
	}

	launcherDir = os.Getenv("VAT_LAUNCHER_DIR")
	if launcherDir == "" {
		Logger.Fatalln("required env VAT_LAUNCHER_DIR is not defined")
	}

	projectName = os.Getenv("VAT_PROJECT_NAME")
	if projectName == "" {
		Logger.Fatalln("required env VAT_PROJECT_NAME is not defined")
	}

	ueVersionCode = os.Getenv("VAT_UE_VERSION_CODE")
	if ueVersionCode == "" {
		Logger.Fatalln("required env VAT_UE_VERSION_CODE is not defined")
	}

	ueVersionMarketplace = os.Getenv("VAT_UE_VERSION_MARKETPLACE")
	if ueVersionMarketplace == "" {
		Logger.Fatalln("required env VAT_UE_VERSION_MARKETPLACE is not defined")
	}

	apiEmail = os.Getenv("VAT_API_EMAIL")
	apiPassword = os.Getenv("VAT_API_PASSWORD")
	if apiEmail == "" && apiPassword == "" {
		Logger.Infof("loading credentials from file")
		ex, err := os.Executable()
		if err != nil {
			Logger.Fatalln(err)
		}
		exPath := filepath.Dir(ex)
		b, err := os.ReadFile(filepath.Join(exPath, ".credentials"))
		if err != nil {
			Logger.Fatalln(err)
		}
		c := string(b)
		t := strings.Split(c, ":")
		apiEmail = t[0]
		apiPassword = t[1]
	}

	rootCmd = &cobra.Command{
		Use: "process",
		Run: func(_ *cobra.Command, args []string) {

			uatPath = os.Getenv("VAT_UAT_PATH")
			if uatPath == "" {
				Logger.Fatalln("required env VAT_UAT_PATH is not defined")
			}

			uvsPath = os.Getenv("VAT_UVS_PATH")
			if uvsPath == "" {
				Logger.Fatalln("required env VAT_UVS_PATH is not defined")
			}

			wailsPath = os.Getenv("VAT_WAILS_PATH")
			if wailsPath == "" {
				Logger.Fatalln("required env VAT_WAILS_PATH is not defined")
			}

			signToolPath = os.Getenv("VAT_SIGNTOOL_PATH")
			if signToolPath == "" {
				Logger.Warningf("optional env VAT_SIGNTOOL_PATH is not defined, app signing will be skipped")
			}

			certFile = os.Getenv("VAT_CERT_FILE")
			if certFile == "" {
				Logger.Warningf("optional env VAT_CERT_FILE is not defined, app signing will be skipped")
			}

			certPassword = os.Getenv("VAT_CERT_PASSWORD")
			if certPassword == "" {
				Logger.Warningf("required env VAT_CERT_PASSWORD is not defined, app signing will be skipped")
			}

			editorPath = os.Getenv("VAT_EDITOR_PATH")
			if editorPath == "" {
				Logger.Fatalln("required env VAT_EDITOR_PATH is not defined")
			}

			platforms = os.Getenv("VAT_PLATFORMS")
			if platforms == "" {
				Logger.Fatalln("required env VAT_PLATFORMS is not defined")
			}

			for _, v := range strings.Split(platforms, ",") {
				supportedPlatforms[v] = true
			}

			jobTypes = os.Getenv("VAT_JOB_TYPES")
			if jobTypes == "" {
				Logger.Fatalln("required env VAT_JOB_TYPES is not defined")
			}

			for _, v := range strings.Split(jobTypes, ",") {
				supportedJobTypes[v] = true
			}

			deployments = os.Getenv("VAT_DEPLOYMENTS")
			if deployments == "" {
				Logger.Fatalln("required env VAT_DEPLOYMENTS is not defined")
			}

			for _, v := range strings.Split(deployments, ",") {
				supportedDeployments[v] = true
			}

			for {
				if err := process(); err != nil {
					Logger.Errorf("error during job processing: %v", err)
				}
			}
		},
	}

	releaseCmd := &cobra.Command{Use: "release", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, args []string) {
		err := pushCodeRelease()
		if err != nil {
			Logger.Errorf("error during release processing: %v", err)
		}
	}}

	rootCmd.AddCommand(releaseCmd)
}

// process Main processing function, fetches the next unclaimed job and runs a corresponding processing function depending on the job type
func process() (err error) {

	//region Wait for an unclaimed job

	var job *JobMetadata
	job, err = fetchUnclaimedJob()
	if job == nil {
		Logger.Infof("waiting for job")
		// Wait before the next request
		time.Sleep(10 * time.Second)
		return nil
	}

	//endregion

	//region Defer job status update

	defer func(job *JobMetadata) {
		if job != nil {
			if err != nil {
				if err1 := updateJobStatus(*job, JobStatusError, err.Error()); err1 != nil {
					Logger.Errorf("failed to update job status: %v", err1)
				}
			} else {
				if err1 := updateJobStatus(*job, JobStatusCompleted, ""); err1 != nil {
					Logger.Errorf("failed to update job status: %v", err1)
				}
			}
		} else {
			Logger.Errorf("failed to update job status, job is nil")
		}
	}(job)

	//endregion

	//region Validation

	// Validate job type
	if !SupportedJobTypes[job.Type] {
		err = fmt.Errorf("unsupported job type: %s", job.Type)
		return
	}

	// Validate job deployment
	if !SupportedJobDeployments[job.Deployment] {
		err = fmt.Errorf("unsupported job deployment: %s", job.Deployment)
		return
	}

	// Validate job platform
	if !supportedPlatforms[job.Platform] {
		err = fmt.Errorf("unsupported job platform: %s", job.Platform)
		return
	}

	//endregion

	//region Process the job

	Logger.Infof("claimed job %s", job.Id)

	if job.Type == "Release" {
		if job.Release == nil {
			return fmt.Errorf("job has no release metadata")
		}

		Logger.Infof("processing release %s %v for %s %s", job.Release.Id, job.Release.Name, job.Platform, job.Deployment)
		if job.Deployment == "Client" {
			err = processClientRelease(*job)
		} else if job.Deployment == "Server" {
			err = processServerRelease(*job)
		} else if job.Deployment == "SDK" {
			err = processSdkRelease(*job)
		} else {
			err = fmt.Errorf("unsupported job deployment %s for type %s", job.Deployment, job.Type)
		}
	} else if job.Type == "Package" {
		if job.Package == nil {
			return fmt.Errorf("job has no package metadata")
		}

		Logger.Infof("processing package %s %s for %s %s", job.Package.Id, job.Package.Name, job.Platform, job.Deployment)
		if job.Deployment == "Client" {
			err = processClientPackage(*job)
		} else if job.Deployment == "Server" {
			err = processServerPackage(*job)
		} else {
			err = fmt.Errorf("unsupported job deployment %s for type %s", job.Deployment, job.Type)
		}
	} else if job.Type == "Launcher" {
		if job.App == nil {
			return fmt.Errorf("job has no app metadata")
		}

		Logger.Infof("processing launcher %s %s for %s %s", job.App.Id, job.App.Name, job.Platform, job.Deployment)
		if job.Deployment == "Client" {
			err = processClientLauncher(*job)
		} else {
			err = fmt.Errorf("unsupported job deployment %s for type %s", job.Deployment, job.Type)
		}
	} else {
		err = fmt.Errorf("unsupported job deployment %s for type %s", job.Deployment, job.Type)
	}

	//endregion

	return err
}

func main() {
	//region Login
	var err error
	token, err = login()
	if err != nil {
		Logger.Fatalf("failed to login: %v", err)
	}
	//endregion

	if err := rootCmd.Execute(); err != nil {
		Logger.Errorf("error executing automation tool: %v", err)
		os.Exit(1)
	}
}
