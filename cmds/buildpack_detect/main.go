package main

import (
	"appd-v3-buildpack/buildpack"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const appVersion = "0.1"

// The flag package provides a default help printer via -h switch
var versionFlag = flag.Bool("verson", false, "Print the version number.")
var logger = log.New(os.Stdout, "appd buildpack detect: ", log.Lshortfile)

func main() {

	flag.Parse() // Scan the arguments list

	if *versionFlag {
		fmt.Println("Version:", appVersion)
	}
	logger.Println("Detecting AppD env vars")
	envVars, varsComplete := buildpack.ParseAppDynamicsEnvVars()
	if !varsComplete {
		logger.Println("Missing AppD env vars detected")
		logger.Printf("Env vars: %p", &envVars)
	}
	fileToDetect := "pom.xml"
	detected, err := detectFile("/tmp", fileToDetect)
	if err != nil {
		logger.Printf("Failed to detect file: %s (%s)", fileToDetect, err.Error())
	}
	logger.Printf("detected: %v\n", detected)
}

func detectFile(dir string, file string) (bool, error) {
	cmd := exec.Command("find", ".", "-name", file)
	cmd.Dir = "/tmp/"
	cmd.Stdout = os.Stdout
	result := cmd.Run()
	logger.Printf("result: %s\n", result)
	return true, nil
}

func detectFile2(dir string, file string) (bool, error) {
	detected := false
	e := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			logger.Println(info.Name())
			if err == nil && file == info.Name() {
				logger.Println("file detected")
				detected = true
				return nil
			}
			return err
		})
	return detected, e
}
