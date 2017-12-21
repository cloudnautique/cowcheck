package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/davecgh/go-spew/spew"
	"github.com/miekg/dns"
	"net/http"
	"time"
	"os"
	"strconv"
	dockerClient "github.com/docker/docker/client"
	"context"
	"strings"
	"github.com/dustin/go-humanize"
  "github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus"
)

var VERSION = "v0.2.0"

var checkSlice = []CheckInterface{}


var dataSpaceFree = uint64(0)
var metadataSpaceFree = uint64(0)

// Primary representation of node health
var nodeHealth = true

// Generics
type Config struct {
	logLevel	logrus.Level
	pollInterval	int
	dataStorageThreshold uint64
	metaDataStorageThreshold uint64
	enableStorageCheck bool
}

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
	cfg			  Config
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

// CheckDNS is a check that looks for a healthy response from the internal DNS zone of Rancher
type CheckDNS struct {
	Check
}

func prometheusHandler() http.Handler {
	return promhttp.Handler()
}

var promNodeHealth = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "cowcheck",
	Subsystem: "node",
	Name:      "cowcheck_node_health",
	Help:      "Boolean representation of overall health of node based on sum of all checks",
})

var promDockerDataStorageFree = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "cowcheck",
	Subsystem: "node",
	Name:      "docker_data_storage",
	Help:      "Amount of free Docker Data Storage space in bytes",
})


var promDockerMetadataStorageFree = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "cowcheck",
	Subsystem: "node",
	Name:      "docker_metadata_storage",
	Help:      "Amount of free Docker Metadata Storage space in bytes",
})

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
			logrus.WithFields(logrus.Fields{"type":"check_results"}).Error(err)
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
	httpClient := http.Client{Timeout: time.Duration(15 * time.Second)}
	resp, err := httpClient.Get("http://169.254.169.250")
	if err != nil {
		logrus.WithFields(logrus.Fields{"type":"check_results"}).Error(err)
		c.fail()
		return true
	}
	defer resp.Body.Close()
	logrus.WithFields(logrus.Fields{"before_eval": "false"}).Debug(spew.Sdump(c))
	return true
}

// CheckStorage

// CheckMetadata is a check for the Metadata Service
type CheckStorage struct {
	Check
}

func NewCheckStorage(cfg Config) *CheckStorage {
	return &CheckStorage{
		Check{
			name:          "CheckStorage",
			description:   "A check for the Docker Storage subsystem",
			currentStatus: true,
			cfg: cfg,
		},
	}
}

func (c *CheckStorage) eval() bool {
	logrus.Infof("Evaluating check %s", c.name)
	logrus.WithFields(logrus.Fields{"before_eval": "true"}).Debug(spew.Sdump(c))

	if c.cfg.enableStorageCheck {
		cli, err := dockerClient.NewEnvClient()
		info, err := cli.Info(context.Background())
		if err != nil {
			panic(err)
		}
		for _, item := range info.DriverStatus {
			if item[0] == "Data Space Available" {
				dataSpaceFree, err = humanize.ParseBytes(item[1])
				if err != nil {
					panic(err)
				}
				promDockerDataStorageFree.Set(float64(dataSpaceFree))
				logrus.Debugf("Found 'Data Space Available' value of ", item[1])
			} else if item[0] == "Metadata Space Available" {
				metadataSpaceFree, err = humanize.ParseBytes(item[1])
				if err != nil {
					panic(err)
				}
				promDockerMetadataStorageFree.Set(float64(metadataSpaceFree))
				logrus.Debugf("Found 'Metadata Space Available' value of ", item[1])
			} else {
				// we didn't find any of our storage keys
				logrus.Error("Didn't find storage information from Docker API. Skipping storage evaluation")
				return true
			}
		}

		if dataSpaceFree < c.cfg.dataStorageThreshold {
			logrus.Errorf("'Data Space Available' is below threshold, failing storage check")
			c.fail()
		}
		if metadataSpaceFree < c.cfg.metaDataStorageThreshold {
			logrus.Errorf("'Metadata Space Available' is below threshold, failing storage check")
			c.fail()
		}
	} else {
		logrus.Debugf("Skipping storage check per user config")
	}



	logrus.WithFields(logrus.Fields{"before_eval": "false"}).Debug(spew.Sdump(c))
	return true
}


// HTTP Server
func checkState(w http.ResponseWriter, r *http.Request) {
	if nodeHealth {
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
	for _, check := range checkSlice {
		logrus.Debugf("checkState - Reading state of check %s", check.getName())
		if check.getStatus() == false {
			nodeHealth = false
			promNodeHealth.Set(1)
		}
	}
}

func checkPoller(checks []CheckInterface, pollInterval int) {
	evalChecks(checks) // call once for instant first tick
	t := time.NewTicker(time.Second * time.Duration(pollInterval))
	for range t.C {
		evalChecks(checks)
	}
}

func parseConfig() Config {
	_logLevel, found := os.LookupEnv("LOG_LEVEL")
	if found != true {
		_logLevel = "WARN"
	}
	logLevel, _ := logrus.ParseLevel(_logLevel)


	_pollInterval, found := os.LookupEnv("POLL_INTERVAL")
	if found != true {
		_pollInterval = "10"
	}
	pollInterval,_ := strconv.Atoi(_pollInterval)

	_dataSpaceThreshold, found := os.LookupEnv("DATA_SPACE_THRESHOLD")
	if found != true {
		_dataSpaceThreshold = "1000"
	}
	dataSpaceThreshold,_ := strconv.ParseUint(_dataSpaceThreshold, 10, 64)

	_metaDataSpaceThreshold, found := os.LookupEnv("METADATA_SPACE_THRESHOLD")
	if found != true {
		_metaDataSpaceThreshold = "1000"
	}
	metaDataSpaceThreshold,_ := strconv.ParseUint(_metaDataSpaceThreshold, 10, 64)

	enableStorageCheck := false
	_enableStorageCheck , found := os.LookupEnv("ENABLE_STORAGE_CHECK")
	if found != true {
		enableStorageCheck = false
	} else {
		if strings.ToLower(_enableStorageCheck) == "true" {
			enableStorageCheck = true
		}
	}

	return Config{
		logLevel,
		pollInterval,
		dataSpaceThreshold,
		metaDataSpaceThreshold,
		enableStorageCheck,

	}

}

func init() {
	prometheus.MustRegister(promNodeHealth, promDockerDataStorageFree, promDockerMetadataStorageFree)
	promNodeHealth.Set(0)
}

func main() {
	cfg := parseConfig()
	logrus.SetLevel(cfg.logLevel)
	logrus.Warn("Starting cowcheck...")
	checkSlice = append(checkSlice, NewCheckDNS(), NewCheckMetadata(), NewCheckStorage(cfg))
	go checkPoller(checkSlice, cfg.pollInterval)

	http.HandleFunc("/", checkState)
	http.HandleFunc("/health", checkState)
	http.Handle("/metrics", prometheusHandler())  // prometheus metrics endpoint

	err := http.ListenAndServe(":5050", nil)
	if err != nil {
		logrus.Error(err)
	}
}
