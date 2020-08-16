package log

import (
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/viper"
)

var (
	logFile *os.File
	Debug   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func SetupLog() func() {
	if logFile == nil {
		file, err := os.OpenFile("runtainer.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		logFile = file
	}

	logFlags := log.Ldate | log.Ltime | log.Lshortfile

	var debugOut io.Writer
	if viper.GetBool("verbose") {
		debugOut = logFile
	} else {
		debugOut = ioutil.Discard
	}
	Debug = log.New(debugOut, "[DEBUG] ", logFlags)
	Info = log.New(logFile, "[INFO] ", logFlags)
	Warning = log.New(logFile, "[WARNING] ", logFlags)
	Error = log.New(logFile, "[ERROR] ", logFlags)

	Debug.Print("Logger initialized")

	return func() {
		logFile.Close()
	}
}
