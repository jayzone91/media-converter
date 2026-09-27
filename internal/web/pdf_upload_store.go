package web

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

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

type pdfUploadStore struct {
	mu sync.Mutex

	uploads map[string]storedPDFUpload

	stop chan struct{}
	done chan struct{}

	closeOnce sync.Once
}

func newPDFUploadStore() *pdfUploadStore {
	store := &pdfUploadStore{
		uploads: make(
			map[string]storedPDFUpload,
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

	s.uploads[id] = upload

	s.mu.Unlock()

	return upload, nil
}

func (s *pdfUploadStore) Get(
	id string,
) (storedPDFUpload, bool) {
	if id == "" {
		return storedPDFUpload{}, false
	}

	s.mu.Lock()

	upload, ok :=
		s.uploads[id]

	if !ok {
		s.mu.Unlock()

		return storedPDFUpload{}, false
	}

	if time.Since(
		upload.CreatedAt,
	) > uploadTTL {
		delete(
			s.uploads,
			id,
		)

		s.mu.Unlock()

		_ = os.RemoveAll(
			upload.Directory,
		)

		return storedPDFUpload{}, false
	}

	s.mu.Unlock()

	return upload, true
}

func (s *pdfUploadStore) Delete(
	id string,
) {
	if id == "" {
		return
	}

	s.mu.Lock()

	upload, ok :=
		s.uploads[id]

	if ok {
		delete(
			s.uploads,
			id,
		)
	}

	s.mu.Unlock()

	if ok {
		_ = os.RemoveAll(
			upload.Directory,
		)
	}
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
	now := time.Now()

	var expired []storedPDFUpload

	s.mu.Lock()

	for id, upload := range s.uploads {
		if now.Sub(
			upload.CreatedAt,
		) <= uploadTTL {
			continue
		}

		delete(
			s.uploads,
			id,
		)

		expired = append(
			expired,
			upload,
		)
	}

	s.mu.Unlock()

	for _, upload := range expired {
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

			for _, upload := range s.uploads {
				uploads = append(
					uploads,
					upload,
				)
			}

			clear(
				s.uploads,
			)

			s.mu.Unlock()

			for _, upload := range uploads {
				_ = os.RemoveAll(
					upload.Directory,
				)
			}
		},
	)
}
