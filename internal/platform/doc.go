// Package platform isolates operating-system integration (native dialogs,
// autostart, DPAPI secrets, shell-open, notifications). Windows is the
// shipping target; *_other.go files provide inert fallbacks so the rest of the
// code base builds and tests on any OS.
package platform
