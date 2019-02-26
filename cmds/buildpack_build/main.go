package main 

import (
    "flag"
    "log"
	"os"
	"appd-v3-buildpack/appdynamics/agent"
	"appd-v3-buildpack/buildpack"	
)

const APP_VERSION = "0.1"

// The flag package provides a default help printer via -h switch
var versionFlag *bool = flag.Bool("version", false, "Print the version number.")

func main() {
  	logger := log.New(os.Stdout, "appd buildpack detect: ", log.Lshortfile)
  
    flag.Parse() // Scan the arguments list 

    if *versionFlag {
        logger.Printf("Version:", APP_VERSION)
    }

	if agentPackagePath, err := agent.DownloadFileFromHttpEnvVar(buildpack.DownloadDir); err != nil {
		logger.Panic(err.Error())
	} else {
		logger.Printf("Downloaded %s from %s\n", agentPackagePath, os.Getenv(agent.EnvHttpAgentDownload))
	}

	if _, err := agent.Unzip(buildpack.AgentPath, buildpack.InstallDir); err != nil {
		logger.Panic(err.Error())		
	}
	envVars, varsComplete := buildpack.ParseAppDynamicsEnvVars()
    if !varsComplete {
    	logger.Println("Missing AppD env vars detected")
    	logger.Printf("Env vars: %p", &envVars)
    }
	buildpack.WriteSetEnvFile(envVars, buildpack.InstallDir, buildpack.SetEnvDir, "app1")
}


