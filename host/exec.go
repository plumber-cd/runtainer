package host

import (
	"os/exec"
	"strings"

	"github.com/plumber-cd/runtainer/log"
)

// Exec exec command on the host and return the output
func Exec(cmd *exec.Cmd) string {
	log.Debug.Printf("Executing: %s", cmd.String())

	out, err := cmd.Output()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			log.Stderr.Print(string(exitErr.Stderr))
		}
		log.Stderr.Panic(err)
	}
	s := string(out)

	log.Debug.Printf("Output: %s", s)
	return strings.TrimSpace(s)
}
