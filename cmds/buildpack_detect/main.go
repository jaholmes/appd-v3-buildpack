package main 

import (
    "flag"
    "log"
    "fmt"
    "os"
    "appd-v3-buildpack/buildpack"
)

const APP_VERSION = "0.1"

// The flag package provides a default help printer via -h switch
var versionFlag *bool = flag.Bool("v", false, "Print the version number.")

func main() {
	logger := log.New(os.Stdout, "appd buildpack detect: ", log.Lshortfile)

    flag.Parse() // Scan the arguments list 

    if *versionFlag {
        fmt.Println("Version:", APP_VERSION)
    }
    logger.Println("Detecting AppD env vars")
    envVars, varsComplete := buildpack.ParseAppDynamicsEnvVars()
    if !varsComplete {
    	logger.Println("Missing AppD env vars detected")
    	logger.Printf("Env vars: %p", &envVars)
    }
}



