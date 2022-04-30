// Package log.
// There are 5 different loggers.
// 4 loggers are disabled by default - Debug, Info, Warning and Error.
// When enabled - they are writing to a file, otherwise they are writing to the void.
// Regular log mode enables Info, Warning and Error, while debug mode enables all 4 of them.
// Separately there is a logger log.Normal which is for main communication with the user.
// The tool never prints to the StdOut reserving that channel exclusively
// to the container in case it's being pipe'd for output processing.
// Thus - all log.Normal messages being print to StdErr.
// Quiet mode will redirect log.Normal into the log.Info logger,
// which is discarded by default.
// All log.Normal messages can be filtered out with regexp `^runtainer\:\s`.
package log

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var (
	logFileOnce sync.Once
	logFile     *os.File
	logWriter   io.Writer
	Debug       *log.Logger
	Info        *log.Logger
	Warning     *log.Logger
	Error       *log.Logger
	Normal      *log.Logger
)

// SetupLog initializes the loggers that are exported by this module.
// Returns a callback function that when called will close any open resources by the loggers, such as files.
// It can be called multiple times, when the logging level settings changes.
// Every instance of a callback function returned can be used and they are equivalent.
func SetupLog() func() {
	// initially, before we read cobra and viper, all logs will remain disabled with no possibility to enable it
	// if we need to debug anything related to cobra/viper routines, at least we can use these env variables to configure loggers from the get go
	var quiet, debug, info bool
	if viper.IsSet("debug") {
		debug = viper.GetBool("debug")
	} else {
		debug = strings.ToLower(os.Getenv("RT_DEBUG")) == "true"
	}
	if viper.IsSet("quiet") {
		quiet = viper.GetBool("quiet")
	} else {
		quiet = strings.ToLower(os.Getenv("RT_QUIET")) == "true"
	}
	if viper.IsSet("log") {
		info = viper.GetBool("log")
	} else {
		info = strings.ToLower(os.Getenv("RT_LOG")) == "true"
	}

	if debug || info {
		logFileOnce.Do(func() {
			var err error
			logFile, err = os.OpenFile("runtainer.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				log.Panic(err)
			}
		})
		logWriter = logFile
	} else {
		logWriter = ioutil.Discard
		if logFile != nil {
			logFile.Close()
		}
	}

	var debugWriter io.Writer
	if debug {
		debugWriter = logWriter
	} else {
		debugWriter = ioutil.Discard
	}

	logFlags := log.Ldate | log.Ltime | log.Lshortfile

	Debug = log.New(debugWriter, "[DEBUG] ", logFlags)
	Info = log.New(logWriter, "[INFO] ", logFlags)
	Warning = log.New(logWriter, "[WARNING] ", logFlags)
	Error = log.New(logWriter, "[ERROR] ", logFlags)

	stderrWriter := io.MultiWriter(os.Stderr, Error.Writer())
	if quiet {
		Normal = Info
	} else {
		Normal = log.New(stderrWriter, "runtainer: ", 0)
	}

	Debug.Print("Logger initialized")

	return func() {
		if logFile != nil {
			Debug.Print("Closing log file")
			logFile.Close()
		}
	}
}
