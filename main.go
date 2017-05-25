package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
	"github.com/miekg/dns"
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
	logrus.Errorf("Check %s has failed", c.name)
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

//CheckDNS is a check for the Kubernetes API
type CheckDNS struct {
	Check
}

func NewCheckDNS() *CheckDNS {
	return &CheckDNS{
		Check{
			name:          "CheckDNS",
			description:   "A check for the DNS Service",
			currentStatus: true,
		},
	}
}

func (c *CheckDNS) eval() bool {
	logrus.Infof("Evaluating check %s", c.name)
	logrus.WithFields(logrus.Fields{"before_eval": "true"}).Debug(spew.Sdump(c))
	c.lastEval = time.Now()

	// borrowing from https://godoc.org/github.com/miekg/dns#example-MX
	config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
	dnsClient := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion("rancher-metadata.rancher.internal.", dns.TypeA)
	m.RecursionDesired = true
	r, _, err := dnsClient.Exchange(m, config.Servers[0]+":"+config.Port)
	if err != nil {
		logrus.Error(err)
		c.fail()
		return true
	}
	if r.Rcode != dns.RcodeSuccess {
		logrus.Error(err)
		c.fail()
		return true
	}

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
	logrus.Warn("Starting cowcheck...")
	checkSlice = append(checkSlice, NewCheckDNS(), NewCheckMetadata())
	go checkPoller(checkSlice)

	http.HandleFunc("/", checkState)
	http.HandleFunc("/health", checkState)

	err := http.ListenAndServe(":5050", nil)
	if err != nil {
		logrus.Error(err)
	}
}
