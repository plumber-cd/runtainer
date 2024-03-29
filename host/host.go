package host

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/plumber-cd/runtainer/log"
	"github.com/spf13/viper"
)

// Host facts about the host
type Host struct {
	Name string
	User string
	UID  int64
	GID  int64
	Home string
	Cwd  string
}

// DiscoverHost discover information about the host
func DiscoverHost() {
	log.Debug.Print("Discover host")

	h := Host{}
	if hst := viper.Get("host"); hst != nil {
		log.Debug.Print("Load user defined host settings")
		// when read from viper for the first time (i.e. nothing Set it there yet as the struct) it will be a map[string]interface{}
		// hence we need to convert it to the struct
		err := mapstructure.Decode(hst, &h)
		if err != nil {
			log.Normal.Panic(err)
		}
	}

	hostName, err := os.Hostname()
	if err != nil {
		log.Normal.Panic(err)
	}
	h.Name = hostName

	currentUser, err := user.Current()
	if err != nil {
		log.Normal.Panic(err)
	}
	h.User = currentUser.Username

	if runtime.GOOS != "windows" {
		log.Debug.Printf("Since the platform is %s, use UID/GID", runtime.GOOS)
		if id, err := strconv.ParseInt(currentUser.Uid, 10, 64); err != nil {
			log.Normal.Panic(err)
		} else {
			h.UID = id
		}
		if id, err := strconv.ParseInt(currentUser.Gid, 10, 64); err != nil {
			log.Normal.Panic(err)
		} else {
			h.GID = id
		}
	}

	home, err := homedir.Dir()
	if err != nil {
		log.Normal.Panic(err)
	}
	h.Home = home

	// What to assume a host cwd when executing container
	if d := viper.GetString("dir"); d != "" {
		log.Debug.Printf("Use user provided cwd %s", d)
		h.Cwd, err = filepath.Abs(d)
		if err != nil {
			log.Normal.Panic(err)
		}
	} else {
		log.Debug.Print("Use actual cwd")

		cwd, err := os.Getwd()
		if err != nil {
			log.Normal.Panic(err)
		}

		h.Cwd = cwd
	}

	log.Debug.Print("Publish to viper")
	viper.Set("host", h)
}
