package githost

import "os/exec"

// Runner runs an external CLI (gh or glab) with the given arguments and
// returns its stdout. ExecRunner is the real implementation; tests supply
// a fake to avoid needing live gh/glab authentication.
type Runner func(name string, args ...string) ([]byte, error)

func ExecRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}
