//go:build !windows

package platform

import (
	"fmt"
	"os"
)

// FatalDialog prints the error on non-Windows development hosts.
func FatalDialog(title, message string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", title, message)
}
