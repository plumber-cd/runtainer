package log

import (
	"io"
	"io/ioutil"
	"log"
	"os"
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
)

func SetupLog() func() {
	if viper.GetBool("verbose") || viper.GetBool("log") {
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
	}

	var debugWriter io.Writer
	if viper.GetBool("verbose") {
		debugWriter = logWriter
	} else {
		debugWriter = ioutil.Discard
	}

	logFlags := log.Ldate | log.Ltime | log.Lshortfile

	Debug = log.New(debugWriter, "[DEBUG] ", logFlags)
	Info = log.New(logWriter, "[INFO] ", logFlags)
	Warning = log.New(logWriter, "[WARNING] ", logFlags)
	Error = log.New(logWriter, "[ERROR] ", logFlags)

	Debug.Print("Logger initialized")

	return func() {
		if logFile != nil {
			Debug.Print("Closing log file")
			logFile.Close()
		}
	}
}
