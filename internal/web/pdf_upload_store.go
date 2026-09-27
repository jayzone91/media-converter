package web

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const pdfUploadProtectionDuration = 35 * time.Minute

type storedPDFUpload struct {
	ID string

	Directory string

	Path string

	PreviewDirectory string

	Filename string

	Size int64

	PageCount int

	CreatedAt time.Time
}

func (u storedPDFUpload) PreviewPath(
	page int,
) string {
	return filepath.Join(
		u.PreviewDirectory,
		fmt.Sprintf(
			"page-%04d.jpg",
			page,
		),
	)
}

type storedPDFUploadEntry struct {
	Upload storedPDFUpload

	ProtectedUntil time.Time

	DeletePending bool
}

type pdfUploadStore struct {
	mu sync.Mutex

	uploads map[string]storedPDFUploadEntry

	stop chan struct{}
	done chan struct{}

	closeOnce sync.Once
}

func newPDFUploadStore() *pdfUploadStore {
	store := &pdfUploadStore{
		uploads: make(
			map[string]storedPDFUploadEntry,
		),

		stop: make(
			chan struct{},
		),

		done: make(
			chan struct{},
		),
	}

	go store.cleanupLoop()

	return store
}

func (s *pdfUploadStore) Add(
	directory string,
	path string,
	previewDirectory string,
	filename string,
	size int64,
	pageCount int,
) (storedPDFUpload, error) {
	id, err := randomUploadID()
	if err != nil {
		return storedPDFUpload{}, err
	}

	upload := storedPDFUpload{
		ID: id,

		Directory: directory,

		Path: path,

		PreviewDirectory: previewDirectory,

		Filename: filename,

		Size: size,

		PageCount: pageCount,

		CreatedAt: time.Now(),
	}

	s.mu.Lock()

	s.uploads[id] =
		storedPDFUploadEntry{
			Upload: upload,
		}

	s.mu.Unlock()

	return upload, nil
}

func (s *pdfUploadStore) Get(
	id string,
) (storedPDFUpload, bool) {
	if id == "" {
		return storedPDFUpload{}, false
	}

	now := time.Now()

	s.mu.Lock()

	entry, ok :=
		s.uploads[id]

	if !ok {
		s.mu.Unlock()

		return storedPDFUpload{}, false
	}

	if entry.DeletePending {
		s.mu.Unlock()

		return storedPDFUpload{}, false
	}

	if now.Sub(
		entry.Upload.CreatedAt,
	) > uploadTTL &&
		!pdfUploadEntryProtected(
			entry,
			now,
		) {
		delete(
			s.uploads,
			id,
		)

		s.mu.Unlock()

		_ = os.RemoveAll(
			entry.Upload.Directory,
		)

		return storedPDFUpload{}, false
	}

	entry.ProtectedUntil =
		now.Add(
			pdfUploadProtectionDuration,
		)

	s.uploads[id] =
		entry

	s.mu.Unlock()

	return entry.Upload,
		true
}

func (s *pdfUploadStore) Delete(
	id string,
) {
	if id == "" {
		return
	}

	now := time.Now()

	s.mu.Lock()

	entry, ok :=
		s.uploads[id]

	if !ok {
		s.mu.Unlock()

		return
	}

	if pdfUploadEntryProtected(
		entry,
		now,
	) {
		entry.DeletePending =
			true

		s.uploads[id] =
			entry

		s.mu.Unlock()

		return
	}

	delete(
		s.uploads,
		id,
	)

	s.mu.Unlock()

	_ = os.RemoveAll(
		entry.Upload.Directory,
	)
}

func (s *pdfUploadStore) cleanupLoop() {
	ticker := time.NewTicker(
		uploadCleanupInterval,
	)

	defer ticker.Stop()
	defer close(s.done)

	for {
		select {
		case <-ticker.C:
			s.cleanupExpired()

		case <-s.stop:
			return
		}
	}
}

func (s *pdfUploadStore) cleanupExpired() {
	s.cleanupAt(
		time.Now(),
	)
}

func (s *pdfUploadStore) cleanupAt(
	now time.Time,
) {
	var removable []storedPDFUpload

	s.mu.Lock()

	for id, entry := range s.uploads {
		if pdfUploadEntryProtected(
			entry,
			now,
		) {
			continue
		}

		expired :=
			now.Sub(
				entry.Upload.CreatedAt,
			) > uploadTTL

		if !entry.DeletePending &&
			!expired {
			continue
		}

		delete(
			s.uploads,
			id,
		)

		removable = append(
			removable,
			entry.Upload,
		)
	}

	s.mu.Unlock()

	removePDFUploadDirectories(
		removable,
	)
}

func pdfUploadEntryProtected(
	entry storedPDFUploadEntry,
	now time.Time,
) bool {
	return entry.ProtectedUntil.After(
		now,
	)
}

func removePDFUploadDirectories(
	uploads []storedPDFUpload,
) {
	for _, upload := range uploads {
		_ = os.RemoveAll(
			upload.Directory,
		)
	}
}

func (s *pdfUploadStore) Close() {
	s.closeOnce.Do(
		func() {
			close(s.stop)

			<-s.done

			s.mu.Lock()

			uploads := make(
				[]storedPDFUpload,
				0,
				len(s.uploads),
			)

			for _, entry := range s.uploads {
				uploads = append(
					uploads,
					entry.Upload,
				)
			}

			clear(
				s.uploads,
			)

			s.mu.Unlock()

			removePDFUploadDirectories(
				uploads,
			)
		},
	)
}
