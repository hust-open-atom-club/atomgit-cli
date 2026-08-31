//go:build windows

package agcmd

import "os"

func terminationSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
