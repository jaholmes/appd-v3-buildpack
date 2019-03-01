package main

import (
	"appd-v3-buildpack/buildpack"
	"flag"
	"log"
	"os"
)

const appVersion = "0.1"

// The flag package provides a default help printer via -h switch
var versionFlag = flag.Bool("version", false, "Print the version number.")

func main() {
	logger := log.New(os.Stdout, "appd buildpack detect: ", log.Lshortfile)

	flag.Parse() // Scan the arguments list

	if *versionFlag {
		logger.Println("Version:", appVersion)
	}

	if agentPackagePath, err := buildpack.DownloadFileFromHttpEnvVar(buildpack.DownloadDir); err != nil {
		logger.Panic(err.Error())
	} else {
		logger.Printf("Downloaded %s from %s\n", agentPackagePath, os.Getenv(buildpack.EnvHttpAgentDownload))
	}
	if _, err := buildpack.Unzip(buildpack.AgentPath, buildpack.InstallDir); err != nil {
		logger.Panic(err.Error())
	}
	envVars, varsComplete := buildpack.ParseAppDynamicsEnvVars()
	if !varsComplete {
		logger.Println("Some required AppD env vars are missing")
		logger.Printf("Env vars: %p", &envVars)
	}
	buildpack.WriteSetEnvFile(envVars, buildpack.InstallDir, buildpack.SetEnvDir, "app2")
}
