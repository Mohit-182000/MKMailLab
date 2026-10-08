// Command mkmaillab is the MKMailLab desktop application: a local SMTP server
// with an inbox UI for inspecting emails sent by applications under
// development.
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/wailsapp/wails/v3/pkg/application"

	"localmail/internal/app"
	"localmail/internal/brand"
	"localmail/internal/config"
	"localmail/internal/logging"
	"localmail/internal/platform"
)

//go:embed all:frontend/dist
var assets embed.FS

type cliFlags struct {
	minimized bool
	dataDir   string
	logLevel  string
}

func parseFlags(args []string) cliFlags {
	var f cliFlags
	fs := flag.NewFlagSet(brand.ExecutableName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&f.minimized, "minimized", false, "start hidden in the system tray")
	fs.StringVar(&f.dataDir, "data-dir", "", "override the data directory")
	fs.StringVar(&f.logLevel, "log-level", "", "debug|info|warn|error")
	// Unknown flags (e.g. injected by tooling) must never prevent startup.
	_ = fs.Parse(args)
	return f
}

func main() {
	os.Exit(run())
}

func run() (code int) {
	flags := parseFlags(os.Args[1:])

	paths, err := config.ResolveFromEnvironment(flags.dataDir, brand.Dev)

	// One-time carry-over of data from the app's previous name (LocalMail).
	var legacy config.LegacyMigration
	var legacyErr error
	if err == nil && flags.dataDir == "" && !paths.Portable && !brand.Dev {
		legacy, legacyErr = config.MigrateLegacyData(os.Getenv("LOCALAPPDATA"), paths.Root)
	}

	if err == nil {
		err = paths.EnsureDirs()
	}
	if err != nil {
		platform.FatalDialog(brand.Name, fmt.Sprintf("%s could not prepare its data directory.\n\n%v", brand.Name, err))
		return 1
	}

	level := slog.LevelInfo
	if brand.Dev {
		level = slog.LevelDebug
	}
	if flags.logLevel != "" {
		_ = level.UnmarshalText([]byte(flags.logLevel))
	}
	var console io.Writer
	if brand.Dev {
		console = os.Stderr
	}
	logs, err := logging.New(logging.Options{Dir: paths.Logs, Level: level, Console: console})
	if err != nil {
		platform.FatalDialog(brand.Name, fmt.Sprintf("%s could not open its log file.\n\n%v", brand.Name, err))
		return 1
	}
	defer logs.Close()
	log := logs.Logger

	defer func() {
		if r := recover(); r != nil {
			log.Error("fatal panic", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			platform.FatalDialog(brand.Name, fmt.Sprintf("%s hit an unexpected error and must close.\n\nDetails were written to:\n%s", brand.Name, logs.File))
			code = 2
		}
	}()

	log.Info("starting",
		"version", brand.Version, "commit", brand.Commit, "dev", brand.Dev,
		"data_dir", paths.Root, "portable", paths.Portable)
	if legacy.Moved {
		log.Info("migrated data from previous app name", "from", legacy.From, "to", legacy.To)
	}
	if legacyErr != nil {
		log.Warn("could not migrate data from previous app name; starting fresh", "err", legacyErr)
	}

	core, err := app.New(context.Background(), app.Options{Paths: paths, Logging: logs})
	if err != nil {
		log.Error("backend initialisation failed", "err", err)
		platform.FatalDialog(brand.Name, fmt.Sprintf("%s could not open its database.\n\n%v\n\nLog file:\n%s", brand.Name, err, logs.File))
		return 1
	}

	instanceID := brand.AppID
	if brand.Dev {
		instanceID += ".dev"
	}

	var mainWindow *application.WebviewWindow
	wailsApp := application.New(application.Options{
		Name:        brand.Name,
		Description: brand.Description,
		Services:    core.Services(),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		// Wails' LogLevel only applies to its default logger, so filter here.
		Logger:   logging.WithMinLevel(logging.Component(log, "wails"), slog.LevelWarn),
		LogLevel: slog.LevelWarn,
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: instanceID,
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				log.Info("second instance launched; focusing existing window")
				if mainWindow != nil {
					mainWindow.Show()
					mainWindow.UnMinimise()
					mainWindow.Focus()
				}
			},
		},
	})

	core.AttachEvents(func(name string, data any) { wailsApp.Event.Emit(name, data) })

	mainWindow = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            brand.Name,
		Width:            1440,
		Height:           900,
		MinWidth:         1024,
		MinHeight:        640,
		InitialPosition:  application.WindowCentered,
		Hidden:           flags.minimized,
		BackgroundColour: application.NewRGB(17, 19, 24),
		DevToolsEnabled:  brand.Dev,
		URL:              "/",
		Windows: application.WindowsWindow{
			Theme: application.SystemDefault,
		},
	})

	if err := wailsApp.Run(); err != nil {
		log.Error("application exited with error", "err", err)
		platform.FatalDialog(brand.Name, fmt.Sprintf("%s failed to start.\n\n%v", brand.Name, err))
		return 1
	}
	log.Info("exited cleanly")
	return 0
}
