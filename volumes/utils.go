package volumes

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/plumber-cd/runtainer/host"
	"github.com/plumber-cd/runtainer/log"
	"github.com/plumber-cd/runtainer/utils"
)

// Resolve ~ into user home.
// This is platform agnostic - always uses slash as a separator.
func resolveTilde(h, p string) string {
	// resolve ~ (ir present) to the actual user home
	if strings.HasPrefix(p, "~") {
		p = strings.TrimPrefix(p, "~")
		p = path.Join(h, p)
	}

	return p
}

func checkLocalDir(h host.Host, dc DiscoveryConfig, path string) (bool, string) {
	path = resolveTilde(h.Home, path)

	if dc.UseParent {
		path = filepath.Dir(path)
	}

	// just in case - get rid of .. and etc
	// do that only for src as filepath uses host file separator
	p, err := filepath.Abs(path)
	if err != nil {
		log.Normal.Panic(err)
	}

	exists, err := utils.OsFs.DirExists(p)
	if err != nil {
		log.Normal.Panic(err)
	}
	if !exists {
		log.Debug.Printf("Volume source %s didn't existed on the host, skipping...", p)
		return false, ""
	}

	return true, p
}
