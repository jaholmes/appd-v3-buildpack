package main 

import (
    "flag"
    "log"
	//"github.com/cloudfoundry/libbuildpack"
	"os"
	"appd-v3-buildpack/appdynamics/agent"
)

const APP_VERSION = "0.1"

// The flag package provides a default help printer via -h switch
var versionFlag *bool = flag.Bool("v", false, "Print the version number.")

func main() {
	var logger = log.New(os.Stdout, "appd buildpack: ", log.Lshortfile)
    var downloadDir string = "/tmp/"
    var installDir string = "/tmp/appd-agent/" 
    var agentPath = downloadDir + "AppServerAgent-4.5.7.25056.zip"   
    
    flag.Parse() // Scan the arguments list 

    if *versionFlag {
        logger.Printf("Version:", APP_VERSION)
    }

	if agentPackagePath, err := agent.DownloadFileFromHttpEnvVar(downloadDir); err != nil {
		logger.Panic(err.Error())
	} else {
		logger.Printf("Downloaded %s from %s\n", agentPackagePath, os.Getenv(agent.EnvHttpAgentDownload))
	}

	if _, err := agent.Unzip(agentPath, installDir); err != nil {
		logger.Panic(err.Error())		
	}
}

