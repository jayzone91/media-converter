package web

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPDFUploadStoreGetProtectsUpload(
	t *testing.T,
) {
	store := newPDFUploadStore()
	defer store.Close()

	directory :=
		createPDFUploadStoreTestDirectory(
			t,
		)

	upload, err := store.Add(
		directory,
		filepath.Join(
			directory,
			"document.pdf",
		),
		filepath.Join(
			directory,
			"preview",
		),
		"document.pdf",
		1024,
		5,
	)

	if err != nil {
		t.Fatalf(
			"add upload: %v",
			err,
		)
	}

	if _, ok := store.Get(
		upload.ID,
	); !ok {
		t.Fatal(
			"expected upload to be available",
		)
	}

	store.cleanupAt(
		time.Now().
			Add(
				uploadTTL +
					time.Minute,
			),
	)

	if _, err := os.Stat(
		directory,
	); err != nil {
		t.Fatalf(
			"protected directory removed: %v",
			err,
		)
	}
}

func TestPDFUploadStoreDeleteDefersProtectedUpload(
	t *testing.T,
) {
	store := newPDFUploadStore()
	defer store.Close()

	directory :=
		createPDFUploadStoreTestDirectory(
			t,
		)

	upload, err := store.Add(
		directory,
		filepath.Join(
			directory,
			"document.pdf",
		),
		filepath.Join(
			directory,
			"preview",
		),
		"document.pdf",
		1024,
		5,
	)

	if err != nil {
		t.Fatalf(
			"add upload: %v",
			err,
		)
	}

	if _, ok := store.Get(
		upload.ID,
	); !ok {
		t.Fatal(
			"expected upload to be available",
		)
	}

	store.Delete(
		upload.ID,
	)

	if _, err := os.Stat(
		directory,
	); err != nil {
		t.Fatalf(
			"protected directory removed: %v",
			err,
		)
	}

	if _, ok := store.Get(
		upload.ID,
	); ok {
		t.Fatal(
			"delete-pending upload must not be returned",
		)
	}
}

func TestPDFUploadStoreCleanupRemovesDeletePendingUpload(
	t *testing.T,
) {
	store := newPDFUploadStore()
	defer store.Close()

	directory :=
		createPDFUploadStoreTestDirectory(
			t,
		)

	upload, err := store.Add(
		directory,
		filepath.Join(
			directory,
			"document.pdf",
		),
		filepath.Join(
			directory,
			"preview",
		),
		"document.pdf",
		1024,
		5,
	)

	if err != nil {
		t.Fatalf(
			"add upload: %v",
			err,
		)
	}

	if _, ok := store.Get(
		upload.ID,
	); !ok {
		t.Fatal(
			"expected upload to be available",
		)
	}

	store.Delete(
		upload.ID,
	)

	store.cleanupAt(
		time.Now().
			Add(
				pdfUploadProtectionDuration +
					time.Minute,
			),
	)

	if _, err := os.Stat(
		directory,
	); !os.IsNotExist(
		err,
	) {
		t.Fatalf(
			"expected directory to be removed, got %v",
			err,
		)
	}
}

func TestPDFUploadStoreCleanupRemovesExpiredUpload(
	t *testing.T,
) {
	store := newPDFUploadStore()
	defer store.Close()

	directory :=
		createPDFUploadStoreTestDirectory(
			t,
		)

	upload, err := store.Add(
		directory,
		filepath.Join(
			directory,
			"document.pdf",
		),
		filepath.Join(
			directory,
			"preview",
		),
		"document.pdf",
		1024,
		5,
	)

	if err != nil {
		t.Fatalf(
			"add upload: %v",
			err,
		)
	}

	store.mu.Lock()

	entry :=
		store.uploads[upload.ID]

	entry.Upload.CreatedAt =
		time.Now().
			Add(
				-uploadTTL -
					time.Minute,
			)

	store.uploads[upload.ID] =
		entry

	store.mu.Unlock()

	store.cleanupExpired()

	if _, err := os.Stat(
		directory,
	); !os.IsNotExist(
		err,
	) {
		t.Fatalf(
			"expected expired directory to be removed, got %v",
			err,
		)
	}
}

func TestPDFUploadStoreGetRejectsExpiredUpload(
	t *testing.T,
) {
	store := newPDFUploadStore()
	defer store.Close()

	directory :=
		createPDFUploadStoreTestDirectory(
			t,
		)

	upload, err := store.Add(
		directory,
		filepath.Join(
			directory,
			"document.pdf",
		),
		filepath.Join(
			directory,
			"preview",
		),
		"document.pdf",
		1024,
		5,
	)

	if err != nil {
		t.Fatalf(
			"add upload: %v",
			err,
		)
	}

	store.mu.Lock()

	entry :=
		store.uploads[upload.ID]

	entry.Upload.CreatedAt =
		time.Now().
			Add(
				-uploadTTL -
					time.Minute,
			)

	store.uploads[upload.ID] =
		entry

	store.mu.Unlock()

	if _, ok := store.Get(
		upload.ID,
	); ok {
		t.Fatal(
			"expected expired upload to be rejected",
		)
	}

	if _, err := os.Stat(
		directory,
	); !os.IsNotExist(
		err,
	) {
		t.Fatalf(
			"expected expired directory to be removed, got %v",
			err,
		)
	}
}

func createPDFUploadStoreTestDirectory(
	t *testing.T,
) string {
	t.Helper()

	directory :=
		t.TempDir()

	path := filepath.Join(
		directory,
		"document.pdf",
	)

	if err := os.WriteFile(
		path,
		[]byte(
			"%PDF-1.7\n",
		),
		0o600,
	); err != nil {
		t.Fatalf(
			"write test PDF: %v",
			err,
		)
	}

	return directory
}
