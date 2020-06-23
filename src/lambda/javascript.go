package lambda

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kafkaesque-io/pubsub-function/src/model"
	"github.com/kafkaesque-io/pubsub-function/src/util"

	log "github.com/sirupsen/logrus"
)

// WorkerSignal is a signal object to pass for channel
type WorkerSignal struct{}

// FunctionInstance is the function worker instance running
type FunctionInstance struct {
	ID        string
	URI       url.URL
	comm      chan *WorkerSignal
	Pid       int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TODO: make the map thread safe
// key is function tenant + name
var functionInstances = make(map[string]FunctionInstance)

var port = 2999

// CreateFnInstance creates function instance
func CreateFnInstance(cfg model.PulsarFunctionConfig) (string, error) {
	// if "linux" != runtime.GOOS {
	switch strings.ToLower(cfg.LanguagePack) {
	case "js", "javascript", "node", "nodejs":
		return StartNodeInstance(cfg)
	default:
		return "", fmt.Errorf("unsupported function language pack %s", cfg.LanguagePack)
	}
}

// StartNodeInstance starts node/javascript instance
func StartNodeInstance(cfg model.PulsarFunctionConfig) (url string, err error) {
	port, err = getPort()
	if err != nil {
		return "", err
	}

	log.Infof("file path %s port %d", cfg.FunctionFilePath, port)
	cmd := exec.Command("node", "../function-pack/js/loader.js", strconv.Itoa(port), cfg.FunctionFilePath)
	if err := cmd.Start(); err != nil {
		return "", err
	}
	log.Infof("command %d", cmd.Process.Pid)

	url = "http://localhost:" + strconv.Itoa(port)
	if err := HealthCheckRetry(url, 3); err != nil {
		return "", err
	}

	return url, nil
}

// TODO: recycle and add more checks
func getPort() (int, error) {
	port++
	if port == 8085 { // do use the local http port
		port++
	}

	if port > 49151 {
		return -1, fmt.Errorf("port pool exhausted")
	}
	return port, nil
}

// GetSourceFilePath gets the directory to source file
func GetSourceFilePath(tenant string) string {
	if tenant == "" {
		tenant = "public"
	}

	path := util.AssignString(os.Getenv("FunctionBaseDir"), "/pulsar/functions") + "/" + tenant
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 777)
	}

	return path
}

// HealthCheck checks the any language pack is running
func HealthCheck(url string) error {
	// Update the headers to allow for SSL redirection
	newRequest, err := http.NewRequest(http.MethodGet, url+"/health", nil)
	if err != nil {
		log.Errorf("make http request url %s error %v", url, err)
		return err
	}
	client := &http.Client{}
	response, err := client.Do(newRequest)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health check fails not ok %d", response.StatusCode)
	}
	return nil
}

// HealthCheckRetry is health check with retry
func HealthCheckRetry(url string, retries int) error {
	for i := 0; i < retries; i++ {
		if err := HealthCheck(url); err != nil {
			num := int64(math.Pow(4, float64(i))) * 100
			time.Sleep(time.Duration(num) * time.Millisecond)
		} else {
			return nil
		}
	}
	return fmt.Errorf("health check %s failed with %d number of retry has been reached", url, retries)
}

func RegisterPulsarConsumer(doc model.PulsarFunctionConfig) {
}
