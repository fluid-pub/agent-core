package version

import (
	"fmt"
	"os"
)

// ExitIfVersionOnly inspects os.Args before flag.Parse. If argv[1] is "-version" or "--version",
// it prints semver (one line) and exits with code 0. Otherwise it returns immediately.
// Pass that agent module's own Version (each slug bumps independently; see package doc).
func ExitIfVersionOnly(semver string) {
	if len(os.Args) < 2 {
		return
	}
	switch os.Args[1] {
	case "-version", "--version":
		fmt.Println(semver)
		os.Exit(0)
	}
}
