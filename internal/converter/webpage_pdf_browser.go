package converter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func webPDFPaperDimensions(
	paperSize string,
) (float64, float64, error) {
	switch strings.ToLower(
		paperSize,
	) {
	case "a4":
		return 8.2677165354,
			11.6929133858,
			nil

	case "letter":
		return 8.5,
			11,
			nil

	default:
		return 0,
			0,
			fmt.Errorf(
				"unsupported paper size: %s",
				paperSize,
			)
	}
}

func findBrowserExecutable() (
	string,
	error,
) {
	if override :=
		strings.TrimSpace(
			os.Getenv(
				"CHROME_BIN",
			),
		); override != "" {
		if path, ok :=
			resolveBrowserExecutable(
				override,
			); ok {
			return path,
				nil
		}
	}

	for _, candidate := range browserExecutableNames() {
		if path, err :=
			exec.LookPath(
				candidate,
			); err == nil {
			return path,
				nil
		}
	}

	for _, candidate := range browserExecutablePaths() {
		if info, err :=
			os.Stat(
				candidate,
			); err == nil &&
			!info.IsDir() {
			return candidate,
				nil
		}
	}

	return "",
		fmt.Errorf(
			"Chrome, Chromium or Edge not found",
		)
}

func resolveBrowserExecutable(
	value string,
) (string, bool) {
	if info, err :=
		os.Stat(
			value,
		); err == nil &&
		!info.IsDir() {
		return value,
			true
	}

	path, err :=
		exec.LookPath(
			value,
		)

	if err != nil {
		return "",
			false
	}

	return path,
		true
}

func browserExecutableNames() []string {
	if runtime.GOOS ==
		"windows" {
		return []string{
			"chrome.exe",
			"msedge.exe",
			"chromium.exe",
			"chrome",
			"msedge",
			"chromium",
		}
	}

	return []string{
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
		"microsoft-edge",
		"microsoft-edge-stable",
		"chrome",
	}
}

func browserExecutablePaths() []string {
	var paths []string

	add :=
		func(
			base string,
			parts ...string,
		) {
			if base == "" {
				return
			}

			paths = append(
				paths,
				filepath.Join(
					append(
						[]string{
							base,
						},
						parts...,
					)...,
				),
			)
		}

	if runtime.GOOS ==
		"windows" {
		for _, base := range []string{
			os.Getenv(
				"LOCALAPPDATA",
			),

			os.Getenv(
				"PROGRAMFILES",
			),

			os.Getenv(
				"PROGRAMFILES(X86)",
			),
		} {
			add(
				base,
				"Google",
				"Chrome",
				"Application",
				"chrome.exe",
			)

			add(
				base,
				"Microsoft",
				"Edge",
				"Application",
				"msedge.exe",
			)
		}
	}

	if runtime.GOOS ==
		"darwin" {
		paths = append(
			paths,

			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",

			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",

			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		)
	}

	return paths
}
