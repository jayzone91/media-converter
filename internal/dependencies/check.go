package dependencies

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const checkTimeout = 10 * time.Second

type Tool struct {
	Name    string
	Path    string
	Version string
}

type definition struct {
	name string

	candidates []string

	versionArg []string

	paths func() []string

	checkVersion bool
}

func CheckAll(
	ctx context.Context,
) ([]Tool, error) {
	definitions := toolDefinitions()

	tools := make(
		[]Tool,
		0,
		len(definitions),
	)

	var missing []string

	for _, definition := range definitions {
		tool, err :=
			checkTool(
				ctx,
				definition,
			)

		if err != nil {
			missing = append(
				missing,
				fmt.Sprintf(
					"%s: %v",
					definition.name,
					err,
				),
			)

			continue
		}

		tools = append(
			tools,
			tool,
		)
	}

	if len(missing) > 0 {
		return tools,
			fmt.Errorf(
				"missing or unusable dependencies:\n- %s",
				strings.Join(
					missing,
					"\n- ",
				),
			)
	}

	return tools, nil
}

func checkTool(
	parent context.Context,
	definition definition,
) (Tool, error) {
	path, err :=
		findExecutable(
			definition,
		)

	if err != nil {
		return Tool{}, err
	}

	version := "installed"

	if definition.checkVersion {
		version, err =
			checkToolVersion(
				parent,
				path,
				definition.versionArg,
			)

		if err != nil {
			return Tool{}, err
		}
	}

	return Tool{
		Name: definition.name,

		Path: path,

		Version: version,
	}, nil
}

func checkToolVersion(
	parent context.Context,
	path string,
	args []string,
) (string, error) {
	ctx, cancel :=
		context.WithTimeout(
			parent,
			checkTimeout,
		)
	defer cancel()

	command :=
		exec.CommandContext(
			ctx,
			path,
			args...,
		)

	output, err :=
		command.CombinedOutput()

	if ctxErr := ctx.Err(); ctxErr != nil {
		return "",
			fmt.Errorf(
				"version check timed out: %w",
				ctxErr,
			)
	}

	if err != nil {
		return "",
			fmt.Errorf(
				"version check failed: %w: %s",
				err,
				strings.TrimSpace(
					string(output),
				),
			)
	}

	version :=
		firstOutputLine(
			string(output),
		)

	if version == "" {
		version = "unknown"
	}

	return version, nil
}

func findExecutable(
	definition definition,
) (string, error) {
	for _, candidate := range definition.candidates {
		path, err :=
			exec.LookPath(
				candidate,
			)

		if err == nil {
			return path, nil
		}
	}

	if definition.paths != nil {
		for _, candidate := range definition.paths() {
			info, err :=
				os.Stat(
					candidate,
				)

			if err == nil &&
				!info.IsDir() {
				return candidate,
					nil
			}
		}
	}

	return "",
		fmt.Errorf(
			"executable not found",
		)
}

func firstOutputLine(
	output string,
) string {
	output =
		strings.ReplaceAll(
			output,
			"\r\n",
			"\n",
		)

	output =
		strings.ReplaceAll(
			output,
			"\r",
			"\n",
		)

	for _, line := range strings.Split(
		output,
		"\n",
	) {
		line =
			strings.TrimSpace(
				line,
			)

		if line != "" {
			return line
		}
	}

	return ""
}

func toolDefinitions() []definition {
	return []definition{
		{
			name: "FFmpeg",

			candidates: []string{
				"ffmpeg",
			},

			versionArg: []string{
				"-version",
			},

			checkVersion: true,
		},
		{
			name: "ffprobe",

			candidates: []string{
				"ffprobe",
			},

			versionArg: []string{
				"-version",
			},

			checkVersion: true,
		},
		{
			name: "ImageMagick",

			candidates: []string{
				"magick",
			},

			versionArg: []string{
				"-version",
			},

			checkVersion: true,
		},
		{
			name: "LibreOffice",

			candidates: []string{
				"soffice",
				"libreoffice",
			},

			versionArg: []string{
				"--version",
			},

			checkVersion: true,
		},
		{
			name: "pdftotext",

			candidates: []string{
				"pdftotext",
			},

			versionArg: []string{
				"-v",
			},

			checkVersion: true,
		},
		{
			name: "pdftoppm",

			candidates: []string{
				"pdftoppm",
			},

			versionArg: []string{
				"-v",
			},

			checkVersion: true,
		},
		{
			name: "Tesseract",

			candidates: []string{
				"tesseract",
			},

			versionArg: []string{
				"--version",
			},

			checkVersion: true,
		},
		{
			name: "qpdf",

			candidates: []string{
				"qpdf",
			},

			versionArg: []string{
				"--version",
			},

			checkVersion: true,
		},
		{
			name: "Ghostscript",

			candidates: ghostscriptCandidates(),

			versionArg: []string{
				"--version",
			},

			checkVersion: true,
		},
		{
			name: "Chromium",

			candidates: browserExecutableNames(),

			paths: browserExecutablePaths,

			/*
				Browser unter Windows niemals für einen
				Version-Check starten.

				Insbesondere Edge kann bei --version ein
				normales Browserfenster öffnen.
			*/
			checkVersion: false,
		},
	}
}

func ghostscriptCandidates() []string {
	if runtime.GOOS ==
		"windows" {
		return []string{
			"gswin64c",
			"gswin32c",
			"gs",
		}
	}

	return []string{
		"gs",
	}
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

	switch runtime.GOOS {
	case "windows":
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

	case "darwin":
		paths = append(
			paths,
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		)
	}

	return paths
}
