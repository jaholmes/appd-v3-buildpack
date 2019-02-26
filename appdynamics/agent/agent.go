package agent

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudfoundry/libbuildpack"
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

const (
	EnvHttpAgentDownload      = "APPD_AGENT_HTTP_URL"
	EnvHttpAgentConfigUrl     = "APPD_CONF_HTTP_URL"
	AppDynamicsInstallDirName = ".appdynamics"
	VendorDirName             = "vendor"
	LinuxEnvExecutableName    = "appd.sh"
	WindowsEnvExecutableName  = "appd.bat"
)

var (
	AppDynamicsPackageDirName        = "appdynamics"
	AppDynamicsAgentConfigurationDir = "conf"
)

type AgentSupplier interface {
	Run() error
}

type AppDPlan struct {
	Credentials AppDCredential `json:"credentials"`
}

type AppDCredential struct {
	ControllerHost   string `json:"host-name"`
	ControllerPort   string `json:"port"`
	SslEnabled       bool   `json:"ssl-enabled"`
	AccountAccessKey string `json:"account-access-key"`
	AccountName      string `json:"account-name"`
}

type VcapApplication struct {
	ApplicationName  string `json:"application_name"`
	ApplicationId    string `json:"application_id"`
	ApplicationSpace string `json:"space_name"`
}

func GetEnvWithDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func ParseAppDynamicsVcapService() (AppDCredential, error) {
	vcapServices := os.Getenv("VCAP_SERVICES")

	services := make(map[string][]AppDPlan)
	if err := json.Unmarshal([]byte(vcapServices), &services); err != nil {
		return AppDCredential{}, errors.New("could not unmarshal VCAP_SERVICES JSON exiting")
	}

	val, pres := services["appdynamics"]
	if !pres {
		return AppDCredential{}, errors.New("service instance of appdynamics not present")
	}
	return val[0].Credentials, nil
}

func ParseVcapApplication() (VcapApplication, error) {
	vcapApplication := os.Getenv("VCAP_APPLICATION")
	application := VcapApplication{}
	if err := json.Unmarshal([]byte(vcapApplication), &application); err != nil {
		return VcapApplication{}, errors.New("could not unmarshal VCAP_APPLICATION JSON")
	}
	return application, nil
}

func PrintDir(dirname string) error {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fmt.Println(filepath.Join(dirname, file.Name()))
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

func DownloadConfFromHttpEnvVar(destDir string, cfgFiles []string) error {
	fmt.Printf("-----> Downloading %v to %v\n", cfgFiles, destDir)

	downloadUrl := os.Getenv(EnvHttpAgentConfigUrl)
	if downloadUrl == "" {
		fmt.Printf("%s env not set, skipping configuration download\n", EnvHttpAgentConfigUrl)
		return nil
	}

	u, err := url.Parse(downloadUrl)
	if err != nil {
		return fmt.Errorf("invalid download Url")
	}
	downloadPath := u.Path

	for _, cfgFile := range cfgFiles {

		u.Path = path.Join(downloadPath, cfgFile)
		fileName := filepath.Join(destDir, path.Base(u.Path))

		fmt.Printf("-----> Downloading %v as %v\n", u.String(), fileName)

		err = DownloadFile(fileName, u.String())
		if err != nil {
			fmt.Printf("-----> Skipping download of %v file\n Reason: %v\n", cfgFile, err)
		}
	}

	return nil
}

func GetAppDynamicsPkgFromVendor(vendorDir, pkgNamePattern string) (string, error) {
	pkgPathPattern := filepath.Join(vendorDir, pkgNamePattern)
	if matches, err := filepath.Glob(pkgPathPattern); err != nil || matches == nil {
		return "", fmt.Errorf("AppDynamics Agent Package %v in not found in vendor directory: %v",
			pkgNamePattern, err)
	} else {
		return matches[0], nil
	}
}

func LocateAppDynamicsFolder(dirName string) (string, error) {
	appdynamicsLocation := ""

	if err := filepath.Walk(dirName, func(path string, info os.FileInfo, err error) error {
		if os.Getenv("BP_DEBUG") != "" {
			fmt.Printf("----->%v\n", path)
		}
		if _, dirName := filepath.Split(path); dirName == AppDynamicsPackageDirName {
			appdynamicsLocation = path
			return nil
		}
		return nil
	}); err != nil {
		return "", err
	}
	return appdynamicsLocation, nil
}

func LocateAppDynamicsConfigurationFolder(dirName string) (string, error) {
	if location, err := LocateAppDynamicsFolder(dirName); err != nil || location == "" {
		return "", err
	} else {
		configDir := filepath.Join(location, AppDynamicsAgentConfigurationDir)
		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			return "", fmt.Errorf("configuration folder not found in %v", location)
		} else {
			return configDir, nil
		}
	}

}

func CopyAgentConfiguration(appdConfigFolder, installDir string) error {
	if _, err := os.Stat(appdConfigFolder); !os.IsNotExist(err) {
		fmt.Printf("-----> Copying %v directory to %v directory\n", appdConfigFolder, installDir)
		if err := libbuildpack.CopyDirectory(appdConfigFolder, installDir); err != nil {
			return err
		}
	} else {
		fmt.Printf("-----> Did not find any %v folder\n", appdConfigFolder)
	}
	return nil
}

func CopyConfigurationOverrides(buildDir, installConfDir string) error {
	if appdConfigFolder, err := LocateAppDynamicsConfigurationFolder(buildDir); appdConfigFolder != "" {
		if err := CopyAgentConfiguration(appdConfigFolder, installConfDir); err != nil {
			return err
		}
	} else {
		fmt.Printf("-----> Did not find AppDynamics Configuration Override folder: %v\n", err)
	}
	return nil
}

func CreateDirs(dirs []string) error {
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("Creating %v directory\n", dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
	}
	return nil
}

func GetAppName(userProvidedEnv string, applicationConfiguration *VcapApplication) string {
	if appName := os.Getenv(userProvidedEnv); appName == "" {
		return fmt.Sprintf("%s:%s", applicationConfiguration.ApplicationSpace,
			applicationConfiguration.ApplicationName)
	} else {
		return appName
	}
}

func GetTierName(userProvidedEnv string, appDynamicsConfiguration *VcapApplication) string {
	if tierName := os.Getenv(userProvidedEnv); tierName == "" {
		return fmt.Sprintf("%s", appDynamicsConfiguration.ApplicationName)
	} else {
		return tierName
	}
}

func GetNodeName(userProvidedEnv string, appDynamicsConfiguration *VcapApplication) string {
	if nodeName := os.Getenv(userProvidedEnv); nodeName == "" {
		applicationName := appDynamicsConfiguration.ApplicationName
		return fmt.Sprintf("%s:%s", applicationName, "$CF_INSTANCE_INDEX")
	} else {
		return nodeName
	}
}
