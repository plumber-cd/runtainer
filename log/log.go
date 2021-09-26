// Package log - by default, all logging is disabled. The notion of this tool is to run something in a container as it was run on the host.
// That means we absolutely can't tamper with stdout/stderr,
// but also polluting a file system with extra files probably won't be such a good idea either.
// But obviously sometimes you need to debug something and need that extra output - so here we have 4 logger levels defined,
// 3 of which (error/warning/info) can be enabled via --log (or config file), and debug one can be enabled via --verbose.
// Verbose mode also enables --logs automatically.
// Additionally, there is an Stderr logger, that is always pointing to both os.Stderr and log.Error writers and always enabled.
// This is so when something unexpected happens and the tool is unable to finish successfully, we can use log.Normal.Fatal or log.Normal.Panic
// to tell the user what happened even if he didn't enable logs, instead of silently crashing which isn't considered a user-friendly practice.
// To have at least something to distinguish output coming from the tool itself from the backends output, log.Normal always prefixed with "runtainer: ".
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
