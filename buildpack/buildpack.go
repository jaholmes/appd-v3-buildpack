package buildpack

import (
    "flag"
    "log"
    "fmt"
	"os"
	"io/ioutil"
)

const (
	DownloadDir string = "/tmp/"
	SetEnvDir string = "/tmp/"	
	SetEnvFile string = SetEnvDir + "appd_setenv.sh"
    InstallDir string = "/tmp/appdynamics/" 
    AgentPath = DownloadDir + "AppServerAgent-4.5.7.25056.zip"   
)

var (
	versionFlag *bool = flag.Bool("v", false, "Print the version number.")
    logger = log.New(os.Stdout, "appd buildpack build: ", log.Lshortfile)
)

func DetectBuildpackType() {}

// todo: add logic to determine if required vars are set
func ParseAppDynamicsEnvVars() (map[string]string, bool) {
	varsComplete := true	
	envVars := make(map[string]string)	
	envVars["APPDYNAMICS_AGENT_ACCOUNT_NAME"] = os.Getenv("APPDYNAMICS_AGENT_ACCOUNT_NAME")
	envVars["APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY"] = os.Getenv("APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY")	
	envVars["APPDYNAMICS_CONTROLLER_SSL_ENABLED"] = os.Getenv("APPDYNAMICS_CONTROLLER_SSL_ENABLED")
	envVars["APPDYNAMICS_CONTROLLER_HOST_NAME"] = os.Getenv("APPDYNAMICS_CONTROLLER_HOST_NAME")
	envVars["APPDYNAMICS_CONTROLLER_PORT"] = os.Getenv("APPDYNAMICS_CONTROLLER_PORT")						
	envVars["APPDYNAMICS_AGENT_APPLICATION_NAME"] = os.Getenv("APPDYNAMICS_AGENT_APPLICATION_NAME")	
	envVars["APPDYNAMICS_AGENT_TIER_NAME"] = os.Getenv("APPDYNAMICS_AGENT_TIER_NAME")	
	envVars["APPDYNAMICS_AGENT_NODE_NAME"] = os.Getenv("APPDYNAMICS_AGENT_NODE_NAME")						

	return envVars, varsComplete
}

func WriteSetEnvFile(envVars map[string]string, installDir string, 
	setEnvDir string, nodeNamePrefix string) error {

	logger.Println("Writing AppDynamics setenv.sh file")
	java_opts := "export JAVA_OPTS=\"-javaagent:%sjavaagent.jar -Dappdynamics.agent.reuse.nodeName=true -Dappdynamics.agent.reuse.nodeName.prefix=%s ${JAVA_OPTS}\""
	fileContents := fmt.Sprintf(java_opts, installDir, nodeNamePrefix)

	for envKey, envVal := range envVars {
		envStr := fmt.Sprintf("%s=%s", envKey, envVal)
		fileContents += "\n" + envStr
	}
	logger.Println("fileContents: " + fileContents)
	bytes := []byte(fileContents)
	if err := ioutil.WriteFile(SetEnvFile, bytes, 0755); err != nil {
		return err
	}
	return nil
}

