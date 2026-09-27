package web

import (
	"os"
	"sync"
	"time"
)

type storedPDFUpload struct {
	ID           string
	Directory    string
	Path         string
	Filename     string
	Size         int64
	PreviewPaths []string
	CreatedAt    time.Time
}

func (u storedPDFUpload) PageCount() int {
	return len(
		u.PreviewPaths,
	)
}

type pdfUploadStore struct {
	mu      sync.Mutex
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
	filename string,
	size int64,
	previewPaths []string,
) (storedPDFUpload, error) {
	id, err := randomUploadID()
	if err != nil {
		return storedPDFUpload{}, err
	}

	upload := storedPDFUpload{
		ID:        id,
		Directory: directory,
		Path:      path,
		Filename:  filename,
		Size:      size,

		PreviewPaths: append(
			[]string(nil),
			previewPaths...,
		),

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

	upload, ok := s.uploads[id]

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

	upload.PreviewPaths = append(
		[]string(nil),
		upload.PreviewPaths...,
	)

	return upload, true
}

func (s *pdfUploadStore) Delete(
	id string,
) {
	if id == "" {
		return
	}

	s.mu.Lock()

	upload, ok := s.uploads[id]

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
