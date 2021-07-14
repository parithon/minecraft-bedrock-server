package minecraft

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/parithon/minecraft-bedrock-daemon/minecraft-bedrock-server/utils"
)

var (
	minecraftnet string         = "https://www.minecraft.net/en-us/download/server/bedrock"
	userAgent    string         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.33 (KHTML, like Gecko) Chrome/90.0.15.212 Safari/537.33"
	dlregx       *regexp.Regexp = regexp.MustCompile(`https://minecraft.azureedge.net/bin-linux/[^"]*`)
	verreg       *regexp.Regexp = regexp.MustCompile(`bedrock-server-(.+).zip`)
	server       *exec.Cmd
	serverStdin  io.WriteCloser
)

func isServerDownloaded() bool {
	if _, err := os.Stat("bedrock-server/version"); os.IsNotExist(err) {
		return false
	}
	return true
}

func currentVersion() (*string, error) {
	var version string

	if !isServerDownloaded() {
		return nil, nil
	}

	if versionbytes, err := os.ReadFile("bedrock-server/version"); err != nil {
		return nil, err
	} else {
		version = string(versionbytes)
	}

	return &version, nil
}

func setHeaderInfo(request *http.Request) {
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("Accept-Language", "en")
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("User-Agent", userAgent)
}

func latestVersion() (*string, error) {
	var request *http.Request
	var response *http.Response
	var downloadUrl string

	if req, err := http.NewRequest("GET", minecraftnet, nil); err != nil {
		return nil, err
	} else {
		request = req
	}

	setHeaderInfo(request)

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			setHeaderInfo(r)
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	if resp, err := client.Do(request); err != nil {
		return nil, err
	} else {
		response = resp
	}

	defer response.Body.Close()

	if body, err := ioutil.ReadAll(response.Body); err != nil {
		return nil, err
	} else {
		downloadUrl = dlregx.FindString(string(body))
	}

	version := verreg.FindStringSubmatch(downloadUrl)[1]

	return &version, nil
}

func checkForUpdates() (bool, error) {
	var lover, onver string

	if ver, err := currentVersion(); err != nil {
		return false, err
	} else {
		if ver == nil {
			lover = "Not Installed"
		} else {
			lover = *ver
		}
	}
	log.Printf("Local version: %s", lover)

	if ver, err := latestVersion(); err != nil {
		return false, err
	} else {
		onver = *ver
	}
	log.Printf("Online version: %s", onver)

	return lover != onver, nil
}

func downloadServer() (*string, error) {
	var request *http.Request
	var response *http.Response
	var downloadUrl, fileName string
	var file *os.File

	log.Println("Gathering the latest version of Minecraft Bedrock Server...")

	if req, err := http.NewRequest("GET", minecraftnet, nil); err != nil {
		return nil, err
	} else {
		request = req
	}

	setHeaderInfo(request)

	client := &http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			setHeaderInfo(r)
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	if res, err := client.Do(request); err != nil {
		return nil, err
	} else {
		response = res
	}

	defer response.Body.Close()

	if body, err := ioutil.ReadAll(response.Body); err != nil {
		return nil, err
	} else {
		downloadUrl = dlregx.FindString(string(body))
	}

	log.Printf("Downloading from: '%s'", downloadUrl)

	version := verreg.FindStringSubmatch(downloadUrl)[1]

	if fileUrl, err := url.Parse(downloadUrl); err != nil {
		return nil, err
	} else {
		path := fileUrl.Path
		segments := strings.Split(path, "/")
		fileName = segments[len(segments)-1]
	}

	if f, err := os.Create(fileName); err != nil {
		os.Remove(fileName)
		return nil, err
	} else {
		file = f
	}

	if resp, err := client.Get(downloadUrl); err != nil {
		os.Remove(fileName)
		return nil, err
	} else {
		response = resp
	}

	defer response.Body.Close()

	if _, err := io.Copy(file, response.Body); err != nil {
		os.Remove(fileName)
		return nil, err
	}

	defer file.Close()

	serverpath := fmt.Sprintf("bedrock-server-%s", version)

	log.Printf("Unzipping latest Minecraft Bedrock Server to '%s'\n", serverpath)

	if _, err := utils.Unzip(fileName, serverpath); err != nil {
		os.Remove(fileName)
		return nil, err
	}

	os.WriteFile(fmt.Sprintf("%s/version", serverpath), []byte(version), 0666)

	if err := os.Remove(fileName); err != nil {
		log.Printf("Failed to remove the zipped file: '%s'", fileName)
	}

	log.Printf("Completed downloading latest Minecraft Bedrock Server version: %s\n", version)

	return &serverpath, nil
}

func symlink(name string) error {
	data := fmt.Sprintf("/data/%s", name)
	app := fmt.Sprintf("bedrock-server-%s", name)

	if _, err := os.Stat("/data"); os.IsNotExist(err) {
		return nil
	}

	if _, err := os.Stat(data); os.IsNotExist(err) { // file does not exist at data location
		if _, err := os.Stat(app); err != nil { // file does exist at app location
			utils.Copy(app, data) // copy the file to the data location
		}
	} else { // file already exists at the data location
		if _, err := os.Stat(app); err != nil { // file exists at app location
			os.Remove(app) // remove the file so we can symlink it in data location
		}
		if err := os.Symlink(data, app); err != nil {
			return err
		}
	}

	return nil
}

func initializeServer(serverPath string) error {
	log.Println("Initializing new Minecraft Bedrock Server...")

	if _, err := os.Stat("bedrock-server"); err != nil {
	} else {
		os.Remove("bedrock-server")
	}

	if err := os.Symlink(serverPath, "bedrock-server"); err != nil {
		return err
	}

	os.Chmod("bedrock-server/bedrock_server", 0755)

	if err := symlink("worlds"); err != nil {
		return err
	}
	if err := symlink("server.properties"); err != nil {
		return err
	}
	if err := symlink("permissions.json"); err != nil {
		return err
	}
	if err := symlink("whitelist.json"); err != nil {
		return err
	}

	return nil
}

func terminate() {
	msg := "shutting down NOW..."
	log.Println(msg)
	serverStdin.Write([]byte(fmt.Sprintf("say %s\n", msg)))
	time.Sleep(time.Second * 5)
	serverStdin.Write([]byte("stop\n"))
	if _, err := server.Process.Wait(); err != nil {
		log.Fatalf("An error occurred while stopping the Minecraft Bedrock Server\n%d", err)
	}
	serverStdin.Close()
	log.Println("Stopped Minecraft Bedrock Server")
	server = nil
	serverStdin = nil
}

func Start() {
	var serverPath *string

	log.Println("Checking for Minecraft Bedrock updates...")

	if updateAvailable, err := checkForUpdates(); err != nil {
		log.Printf("An error occurred while gathering the Minecraft Bedrock Server versions.\n%d", err)
	} else {
		log.Printf("Update available: %v\n", updateAvailable)
		if updateAvailable {
			if servpath, err := downloadServer(); err != nil {
				log.Printf("An error occurred while downloading the latest Minecraft Bedrock Server.\n%d", err)
				return
			} else {
				serverPath = servpath
			}
		}
	}

	if nil != serverPath {
		if err := initializeServer(*serverPath); err != nil {
			log.Printf("An error occurred while initializing Minecraft Bedrock Server.\n%d", err)
			return
		}
	}

	log.Printf("Starting Minecraft Bedrock Server...")
	server = exec.Command("./bedrock_server")
	server.Dir = "bedrock-server"
	server.Stdout = log.Writer()
	server.Stderr = log.Writer()

	var err error = nil
	serverStdin, err = server.StdinPipe()
	if err != nil {
		log.Fatalf("An error occurred while redirecting Stdin\n%d", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("An error occurred while starting the Minecraft Bedrock Server\n%d", err)
	}

	log.Println("Started Minecraft Bedrock Server")
}

func Stop(signal os.Signal) {
	if signal == syscall.SIGQUIT {
		for i := 6; i > 0; i-- {
			msg := fmt.Sprintf("shutting down in %d seconds...", (i * 5))
			log.Println(msg)
			serverStdin.Write([]byte(fmt.Sprintf("say %s\n", msg)))
			time.Sleep(time.Second * 5)
		}
	}
	terminate()
}
