//go:build windows

package platform

import "golang.org/x/sys/windows"

const (
	mbOK            = 0x00000000
	mbIconError     = 0x00000010
	mbSetForeground = 0x00010000
)

// FatalDialog shows a blocking native error box. It is used only for errors
// that happen before the UI exists (e.g. the data directory is unwritable).
func FatalDialog(title, message string) {
	t, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	m, err := windows.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	_, _ = windows.MessageBox(0, m, t, mbOK|mbIconError|mbSetForeground)
}
