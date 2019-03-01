package buildpack

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Constants that are available outside of this package
const (
	DownloadDir          string = "/tmp/"
	SetEnvDir            string = "/tmp/"
	SetEnvFile           string = SetEnvDir + "appd_setenv.sh"
	InstallDir           string = "/tmp/appdynamics/"
	AgentPath                   = DownloadDir + "AppServerAgent-4.5.7.25056.zip"
	EnvHttpAgentDownload string = "APPD_AGENT_HTTP_URL"
)

var (
	versionFlag *bool = flag.Bool("v", false, "Print the version number.")
	logger            = log.New(os.Stdout, "appd buildpack build: ", log.Lshortfile)
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
	envVars["APPDYNAMICS_NODE_PREFIX"] = os.Getenv("APPDYNAMICS_NODE_PREFIX")

	return envVars, varsComplete
}

func WriteSetEnvFile(envVars map[string]string, installDir string,
	setEnvDir string, nodeNamePrefix string) error {

	logger.Println("Writing AppDynamics setenv.sh")
	javaOpts := "export JAVA_OPTS=\"-javaagent:%sjavaagent.jar -Dappdynamics.agent.reuse.nodeName=true -Dappdynamics.agent.reuse.nodeName.prefix=%s ${JAVA_OPTS}\""
	fileContents := fmt.Sprintf(javaOpts, installDir, envVars["APPDYNAMICS_NODE_PREFIX"])

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

func DownloadFile(filepath string, url string) error {
	resp, err := http.Get(url)

	if err != nil {
		return fmt.Errorf("downloading %v failed with err: %v", url, err)
	}

	if resp == nil {
		return fmt.Errorf("downloading %v failed resp: nil", url)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %v failed\n Status Code: %v\n", url, resp.StatusCode)
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func Unzip(src string, dest string) ([]string, error) {

	var fileNames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return fileNames, err
	}
	defer r.Close()

	for _, f := range r.File {

		rc, err := f.Open()
		if err != nil {
			return fileNames, err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fileNames, fmt.Errorf("%s: illegal file path", fpath)
		}

		fileNames = append(fileNames, fpath)

		if f.FileInfo().IsDir() {

			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)

		} else {

			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return fileNames, err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return fileNames, err
			}

			_, err = io.Copy(outFile, rc)

			// Close the file without defer to close before next iteration of loop
			outFile.Close()

			if err != nil {
				return fileNames, err
			}

		}
	}
	return fileNames, nil
}

func DownloadFileFromHttpEnvVar(destDir string) (string, error) {
	if downloadUrl := os.Getenv(EnvHttpAgentDownload); downloadUrl != "" {
		if u, err := url.Parse(downloadUrl); err != nil {
			return "", errors.New("invalid download Url")
		} else {
			fileName := filepath.Join(destDir, path.Base(u.Path))
			return fileName, DownloadFile(fileName, downloadUrl)
		}
	}

	return "", fmt.Errorf("%s env not set, skipping http download", EnvHttpAgentDownload)
}
