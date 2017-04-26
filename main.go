package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
	"net/http"
	"time"
)

var VERSION = "v0.0.0-dev"

var checkSlice = []CheckInterface{}

// Generics

// CheckInterface is a interface for Checks
type CheckInterface interface {
	eval() bool
	fail() bool
	getStatus() bool
	getName() string
}

type Check struct {
	name          string
	description   string
	lastEval      time.Time
	lastFail      time.Time
	currentStatus bool
}

func (c *Check) eval() bool {
	return true
}

func (c *Check) fail() bool {
	logrus.Infof("Check %s has failed", c.name)
	c.lastFail = time.Now()
	c.currentStatus = false
	return true
}

func (c *Check) getStatus() bool {
	return c.currentStatus
}

func (c *Check) getName() string {
	return c.name
}

// Implemented checks

//CheckKubeAPI is a check for the Kubernetes API
type CheckKubeAPI struct {
	Check
}

func NewCheckKubeAPI() *CheckKubeAPI {
	return &CheckKubeAPI{
		Check{
			name:          "CheckKubeAPI",
			description:   "A check for the Kubernetes API",
			currentStatus: true,
		},
	}
}

func (c *CheckKubeAPI) eval() bool {
	logrus.Infof("Evaluating check %s", c.name)
	logrus.WithFields(logrus.Fields{"before_eval": "true"}).Debug(spew.Sdump(c))
	c.lastEval = time.Now()
	httpClient := http.Client{Timeout: time.Duration(2 * time.Second)}
	resp, err := httpClient.Get("http://kubernetes.kubernetes.rancher.internal")
	if err != nil {
		c.fail()
		return true
	}
	defer resp.Body.Close()
	logrus.WithFields(logrus.Fields{"before_eval": "false"}).Debug(spew.Sdump(c))
	c.currentStatus = true
	return true
}

// CheckMetadata is a check for the Metadata Service
type CheckMetadata struct {
	Check
}

func NewCheckMetadata() *CheckMetadata {
	return &CheckMetadata{
		Check{
			name:          "CheckMetadata",
			description:   "A check for the CheckMetadata Service",
			currentStatus: true,
		},
	}
}

func (c *CheckMetadata) eval() bool {
	logrus.Infof("Evaluating check %s", c.name)
	logrus.WithFields(logrus.Fields{"before_eval": "true"}).Debug(spew.Sdump(c))
	c.lastEval = time.Now()
	httpClient := http.Client{Timeout: time.Duration(2 * time.Second)}
	resp, err := httpClient.Get("http://169.254.169.250")
	if err != nil {
		logrus.Error("Fail")
		c.fail()
		return true
	}
	defer resp.Body.Close()
	logrus.WithFields(logrus.Fields{"before_eval": "false"}).Debug(spew.Sdump(c))
	return true
}

// HTTP Server
func checkState(w http.ResponseWriter, r *http.Request) {
	health := true
	for _, check := range checkSlice {
		logrus.Debugf("checkState - Reading state of check %s", check.getName())
		if check.getStatus() == false {
			health = false
		}
	}
	if health {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Everything OK"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Failed"))
	}
}

func evalChecks(checks []CheckInterface) {
	for _, check := range checks {
		check.eval()
	}
}

func checkPoller(checks []CheckInterface) {
	evalChecks(checks) // call once for instant first tick
	t := time.NewTicker(2 * time.Second)
	for _ = range t.C {
		evalChecks(checks)
	}
}

func main() {
	logrus.SetLevel(logrus.WarnLevel)
	logrus.Info("Starting cowcheck...")
	checkSlice = append(checkSlice, NewCheckKubeAPI(), NewCheckMetadata())
	go checkPoller(checkSlice)

	http.HandleFunc("/", checkState)
	err := http.ListenAndServe(":5050", nil)
	if err != nil {
		logrus.Error(err)
	}
}
