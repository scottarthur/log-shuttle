package main

import (
	"flag"
	"github.com/pebbe/util"
	"log"
	"net/url"
	"os"
	"time"
)

var LogplexUrl = os.Getenv("LOGPLEX_URL")

const (
	INPUT_FORMAT_RAW     = iota
	INPUT_FORMAT_RFC3164 = iota
)

const (
	DEFAULT_INPUT_FORMAT   = INPUT_FORMAT_RAW
	DEFAULT_FRONT_BUFF     = 1000
	DEFAULT_STATS_BUFF     = 5000
	DEFAULT_STATS_ADDR     = ""
	DEFAULT_TIMEOUT        = 5 * time.Second
	DEFAULT_WAIT_DURATION  = 250 * time.Millisecond
	DEFAULT_MAX_ATTEMPTS   = 3
	DEFAULT_STATS_INTERVAL = 0 * time.Second
	DEFAULT_STATS_SOURCE   = ""
)

type ShuttleConfig struct {
	FrontBuff     int
	StatsBuff     int
	BatchSize     int
	NumBatchers   int
	NumOutlets    int
	InputFormat   int
	MaxAttempts   int
	LogsURL       string
	Prival        string
	Version       string
	Procid        string
	Hostname      string
	Appname       string
	Msgid         string
	StatsAddr     string
	StatsSource   string
	SkipHeaders   bool
	SkipVerify    bool
	PrintVersion  bool
	Verbose       bool
	LogToSyslog   bool
	WaitDuration  time.Duration
	Timeout       time.Duration
	StatsInterval time.Duration
}

func (c *ShuttleConfig) ParseFlags() {
	flag.BoolVar(&c.PrintVersion, "version", false, "Print log-shuttle version.")
	flag.BoolVar(&c.Verbose, "verbose", false, "Enable verbose debug info.")
	flag.BoolVar(&c.SkipHeaders, "skip-headers", false, "Skip the prepending of rfc5424 headers.")
	flag.BoolVar(&c.SkipVerify, "skip-verify", false, "Skip the verification of HTTPS server certificate.")
	flag.StringVar(&c.Prival, "prival", "190", "The primary value of the rfc5424 header.")
	flag.StringVar(&c.Version, "syslog-version", "1", "The version of syslog.")
	flag.StringVar(&c.Procid, "procid", "shuttle", "The procid field for the syslog header.")
	flag.StringVar(&c.Appname, "appname", "", "The app-name field for the syslog header.")
	flag.StringVar(&c.Appname, "logplex-token", "", "Secret logplex token.")
	flag.StringVar(&c.Hostname, "hostname", "shuttle", "The hostname field for the syslog header.")
	flag.StringVar(&c.Msgid, "msgid", "- -", "The msgid field for the syslog header.")
	flag.StringVar(&c.LogsURL, "logs-url", "", "The receiver of the log data.")
	flag.StringVar(&c.StatsAddr, "stats-addr", DEFAULT_STATS_ADDR, "Where to expose stats.")
	flag.StringVar(&c.StatsSource, "stats-source", DEFAULT_STATS_SOURCE, "When emitting stats, add source=<stats-source> to the stats.")
	flag.DurationVar(&c.StatsInterval, "stats-interval", time.Duration(DEFAULT_STATS_INTERVAL), "How often to emit/reset stats.")
	flag.IntVar(&c.MaxAttempts, "max-attempts", DEFAULT_MAX_ATTEMPTS, "Max number of retries.")
	flag.IntVar(&c.InputFormat, "input-format", DEFAULT_INPUT_FORMAT, "0=raw (default), 1=rfc3164 (syslog(3))")
	flag.IntVar(&c.NumBatchers, "num-batchers", 2, "The number of batchers to run.")
	flag.IntVar(&c.NumOutlets, "num-outlets", 4, "The number of outlets to run.")
	flag.DurationVar(&c.WaitDuration, "wait", time.Duration(DEFAULT_WAIT_DURATION), "Duration to wait to flush messages to logplex")
	flag.IntVar(&c.BatchSize, "batch-size", 500, "Number of messages to pack into a logplex http request.")
	flag.IntVar(&c.FrontBuff, "front-buff", DEFAULT_FRONT_BUFF, "Number of messages to buffer in log-shuttle's input chanel.")
	flag.IntVar(&c.StatsBuff, "stats-buff", DEFAULT_STATS_BUFF, "Number of stats to buffer.")
	flag.DurationVar(&c.Timeout, "timeout", time.Duration(DEFAULT_TIMEOUT), "Duration to wait for a response from Logplex.")
	flag.BoolVar(&c.LogToSyslog, "log-to-syslog", false, "Log to syslog instead of stderr")
	flag.Parse()

	if c.MaxAttempts < 1 {
		log.Fatalf("-max-attempts must be >= 1")
	}
}

func (c *ShuttleConfig) OutletURL() string {
	var err error
	var oUrl *url.URL

	if len(c.LogsURL) > 0 {
		oUrl, err = url.Parse(c.LogsURL)
		if err != nil {
			log.Fatalf("Unable to parse logs-url")
		}
	}
	if len(LogplexUrl) > 0 {
		oUrl, err = url.Parse(LogplexUrl)
		if err != nil {
			log.Fatalf("Unable to parse $LOGPLEX_URL")
		}
	}

	if oUrl == nil {
		log.Fatalf("Must set -logs-url or $LOGPLEX_URL.")
	}

	if oUrl.User == nil {
		oUrl.User = url.UserPassword("token", c.Appname)
	}
	return oUrl.String()
}

func (c *ShuttleConfig) UseStdin() bool {
	return !util.IsTerminal(os.Stdin)
}
