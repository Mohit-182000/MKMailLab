// Package brand holds every product-identity constant in one place so the
// application can be renamed or rebranded by editing a single file (plus the
// build metadata in build/config.yml and build/windows/info.json).
package brand

const (
	// Name is the human-facing product name.
	Name = "LocalMail"

	// Description is the short product description used in window titles,
	// installers and the About screen.
	Description = "Local SMTP email testing tool"

	// AppID is the reverse-DNS identifier. It is used for the single-instance
	// lock and must stay stable across releases.
	AppID = "dev.localmail.app"

	// AppUserModelID identifies the app to the Windows shell (taskbar grouping,
	// toast notifications). The installer's Start Menu shortcut must use the
	// same value.
	AppUserModelID = "LocalMail.LocalMail"

	// DataDirName is the folder created under %LOCALAPPDATA%.
	DataDirName = "LocalMail"

	// ExecutableName is the base name of the built binary (without .exe).
	ExecutableName = "localmail"

	// PortableMarker is the file name that, when present next to the
	// executable, switches the app into portable mode.
	PortableMarker = "localmail.portable"
)

// Version and Commit are overridden at build time via
//
//	-ldflags "-X localmail/internal/brand.Version=1.2.3 -X localmail/internal/brand.Commit=abc123"
var (
	Version = "0.1.0-dev"
	Commit  = "unknown"
)
