package web

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jayzone91/media-converter/internal/media"
)

func TestServerShutdownCleansTemporaryStores(
	t *testing.T,
) {
	t.Parallel()

	server :=
		NewServer(
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		)

	uploadDirectory :=
		createShutdownTestDirectory(
			t,
			"upload",
		)

	uploadFile :=
		filepath.Join(
			uploadDirectory,
			"input.txt",
		)

	writeShutdownTestFile(
		t,
		uploadFile,
	)

	_, err :=
		server.uploads.Add(
			uploadDirectory,
			[]storedUploadFile{
				{
					Path: uploadFile,

					Filename: "input.txt",
				},
			},
			media.Formats["txt"],
		)

	if err != nil {
		t.Fatalf(
			"add upload: %v",
			err,
		)
	}

	pdfDirectory :=
		createShutdownTestDirectory(
			t,
			"pdf-upload",
		)

	pdfPath :=
		filepath.Join(
			pdfDirectory,
			"document.pdf",
		)

	writeShutdownTestFile(
		t,
		pdfPath,
	)

	previewDirectory :=
		filepath.Join(
			pdfDirectory,
			"previews",
		)

	if err :=
		os.MkdirAll(
			previewDirectory,
			0700,
		); err != nil {
		t.Fatalf(
			"create preview directory: %v",
			err,
		)
	}

	_, err =
		server.pdfUploads.Add(
			pdfDirectory,
			pdfPath,
			previewDirectory,
			"document.pdf",
			1,
			1,
		)

	if err != nil {
		t.Fatalf(
			"add PDF upload: %v",
			err,
		)
	}

	downloadSourceDirectory :=
		t.TempDir()

	downloadSource :=
		filepath.Join(
			downloadSourceDirectory,
			"result.pdf",
		)

	writeShutdownTestFile(
		t,
		downloadSource,
	)

	download, err :=
		server.downloads.Add(
			downloadSource,
			"result.pdf",
		)

	if err != nil {
		t.Fatalf(
			"add download: %v",
			err,
		)
	}

	requireShutdownTestPathExists(
		t,
		uploadDirectory,
	)

	requireShutdownTestPathExists(
		t,
		pdfDirectory,
	)

	requireShutdownTestPathExists(
		t,
		download.Directory,
	)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

	defer cancel()

	if err :=
		server.Shutdown(
			ctx,
		); err != nil {
		t.Fatalf(
			"server shutdown: %v",
			err,
		)
	}

	requireShutdownTestPathRemoved(
		t,
		uploadDirectory,
	)

	requireShutdownTestPathRemoved(
		t,
		pdfDirectory,
	)

	requireShutdownTestPathRemoved(
		t,
		download.Directory,
	)
}

func TestServerShutdownIsIdempotent(
	t *testing.T,
) {
	t.Parallel()

	server :=
		NewServer(
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		)

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			5*time.Second,
		)

	defer cancel()

	if err :=
		server.Shutdown(
			ctx,
		); err != nil {
		t.Fatalf(
			"first shutdown: %v",
			err,
		)
	}

	if err :=
		server.Shutdown(
			ctx,
		); err != nil {
		t.Fatalf(
			"second shutdown: %v",
			err,
		)
	}
}

func createShutdownTestDirectory(
	t *testing.T,
	name string,
) string {
	t.Helper()

	directory :=
		filepath.Join(
			t.TempDir(),
			name,
		)

	if err :=
		os.MkdirAll(
			directory,
			0700,
		); err != nil {
		t.Fatalf(
			"create test directory %q: %v",
			name,
			err,
		)
	}

	return directory
}

func writeShutdownTestFile(
	t *testing.T,
	path string,
) {
	t.Helper()

	if err :=
		os.WriteFile(
			path,
			[]byte(
				"shutdown cleanup test",
			),
			0600,
		); err != nil {
		t.Fatalf(
			"write test file %q: %v",
			path,
			err,
		)
	}
}

func requireShutdownTestPathExists(
	t *testing.T,
	path string,
) {
	t.Helper()

	if _, err :=
		os.Stat(
			path,
		); err != nil {
		t.Fatalf(
			"expected path %q to exist: %v",
			path,
			err,
		)
	}
}

func requireShutdownTestPathRemoved(
	t *testing.T,
	path string,
) {
	t.Helper()

	_, err :=
		os.Stat(
			path,
		)

	if err == nil {
		t.Fatalf(
			"expected path %q to be removed",
			path,
		)
	}

	if !os.IsNotExist(
		err,
	) {
		t.Fatalf(
			"inspect removed path %q: %v",
			path,
			err,
		)
	}
}
