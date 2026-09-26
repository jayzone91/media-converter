package web

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"time"

	"github.com/jayzone91/media-converter/internal/media"
)

const (
	uploadTTL             = 30 * time.Minute
	uploadCleanupInterval = 5 * time.Minute
)

type storedUpload struct {
	ID        string
	Directory string
	Path      string
	Filename  string
	Format    media.Format
	CreatedAt time.Time
}

type uploadStore struct {
	mu        sync.Mutex
	uploads   map[string]storedUpload
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

func newUploadStore() *uploadStore {
	store := &uploadStore{
		uploads: make(map[string]storedUpload),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}

	go store.cleanupLoop()

	return store
}

func (s *uploadStore) Add(
	directory string,
	path string,
	filename string,
	format media.Format,
) (storedUpload, error) {
	id, err := randomUploadID()
	if err != nil {
		return storedUpload{}, err
	}

	upload := storedUpload{
		ID:        id,
		Directory: directory,
		Path:      path,
		Filename:  filename,
		Format:    format,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.uploads[id] = upload
	s.mu.Unlock()

	return upload, nil
}

func (s *uploadStore) Take(
	id string,
) (storedUpload, bool) {
	s.mu.Lock()

	upload, ok := s.uploads[id]
	if !ok {
		s.mu.Unlock()
		return storedUpload{}, false
	}

	delete(s.uploads, id)

	s.mu.Unlock()

	if time.Since(upload.CreatedAt) > uploadTTL {
		_ = os.RemoveAll(upload.Directory)

		return storedUpload{}, false
	}

	return upload, true
}

func (s *uploadStore) Delete(id string) {
	if id == "" {
		return
	}

	s.mu.Lock()

	upload, ok := s.uploads[id]
	if ok {
		delete(s.uploads, id)
	}

	s.mu.Unlock()

	if ok {
		_ = os.RemoveAll(upload.Directory)
	}
}

func (s *uploadStore) cleanupLoop() {
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

func (s *uploadStore) cleanupExpired() {
	now := time.Now()

	var expired []storedUpload

	s.mu.Lock()

	for id, upload := range s.uploads {
		if now.Sub(upload.CreatedAt) > uploadTTL {
			delete(s.uploads, id)

			expired = append(
				expired,
				upload,
			)
		}
	}

	s.mu.Unlock()

	for _, upload := range expired {
		_ = os.RemoveAll(upload.Directory)
	}
}

func (s *uploadStore) Close() {
	s.closeOnce.Do(func() {
		close(s.stop)
		<-s.done

		s.mu.Lock()

		uploads := make(
			[]storedUpload,
			0,
			len(s.uploads),
		)

		for _, upload := range s.uploads {
			uploads = append(
				uploads,
				upload,
			)
		}

		clear(s.uploads)

		s.mu.Unlock()

		for _, upload := range uploads {
			_ = os.RemoveAll(
				upload.Directory,
			)
		}
	})
}

func randomUploadID() (string, error) {
	buffer := make([]byte, 32)

	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}
