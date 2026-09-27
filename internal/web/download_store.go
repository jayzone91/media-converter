package web

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	downloadTTL             = 10 * time.Minute
	downloadCleanupInterval = 2 * time.Minute
)

type storedDownload struct {
	ID        string
	Directory string
	Path      string
	Filename  string
	CreatedAt time.Time
}

type downloadStore struct {
	mu sync.Mutex

	downloads map[string]storedDownload

	stop chan struct{}
	done chan struct{}

	closeOnce sync.Once
}

func newDownloadStore() *downloadStore {
	store := &downloadStore{
		downloads: make(
			map[string]storedDownload,
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

func (s *downloadStore) Add(
	sourcePath string,
	filename string,
) (storedDownload, error) {
	id, err := randomUploadID()
	if err != nil {
		return storedDownload{}, err
	}

	directory, err := os.MkdirTemp(
		"",
		"media-converter-download-*",
	)
	if err != nil {
		return storedDownload{},
			fmt.Errorf(
				"create download directory: %w",
				err,
			)
	}

	cleanup := true

	defer func() {
		if cleanup {
			_ = os.RemoveAll(
				directory,
			)
		}
	}()

	filename = filepath.Base(
		filename,
	)

	if filename == "." ||
		filename == "" {
		return storedDownload{},
			fmt.Errorf(
				"invalid download filename",
			)
	}

	destinationPath := filepath.Join(
		directory,
		filename,
	)

	if err := moveDownloadFile(
		sourcePath,
		destinationPath,
	); err != nil {
		return storedDownload{},
			err
	}

	download := storedDownload{
		ID: id,

		Directory: directory,

		Path: destinationPath,

		Filename: filename,

		CreatedAt: time.Now(),
	}

	s.mu.Lock()

	s.downloads[id] =
		download

	s.mu.Unlock()

	cleanup = false

	return download, nil
}

func (s *downloadStore) Take(
	id string,
) (storedDownload, bool) {
	if id == "" {
		return storedDownload{}, false
	}

	s.mu.Lock()

	download, ok :=
		s.downloads[id]

	if ok {
		delete(
			s.downloads,
			id,
		)
	}

	s.mu.Unlock()

	if !ok {
		return storedDownload{}, false
	}

	if time.Since(
		download.CreatedAt,
	) > downloadTTL {
		_ = os.RemoveAll(
			download.Directory,
		)

		return storedDownload{}, false
	}

	return download, true
}

func (s *downloadStore) cleanupLoop() {
	ticker := time.NewTicker(
		downloadCleanupInterval,
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

func (s *downloadStore) cleanupExpired() {
	now := time.Now()

	var expired []storedDownload

	s.mu.Lock()

	for id, download := range s.downloads {
		if now.Sub(
			download.CreatedAt,
		) <= downloadTTL {
			continue
		}

		delete(
			s.downloads,
			id,
		)

		expired = append(
			expired,
			download,
		)
	}

	s.mu.Unlock()

	for _, download := range expired {
		_ = os.RemoveAll(
			download.Directory,
		)
	}
}

func (s *downloadStore) Close() {
	s.closeOnce.Do(
		func() {
			close(s.stop)

			<-s.done

			s.mu.Lock()

			downloads := make(
				[]storedDownload,
				0,
				len(s.downloads),
			)

			for _, download := range s.downloads {
				downloads = append(
					downloads,
					download,
				)
			}

			clear(
				s.downloads,
			)

			s.mu.Unlock()

			for _, download := range downloads {
				_ = os.RemoveAll(
					download.Directory,
				)
			}
		},
	)
}

func moveDownloadFile(
	source string,
	destination string,
) error {
	if err := os.Rename(
		source,
		destination,
	); err == nil {
		return nil
	}

	sourceFile, err := os.Open(
		source,
	)
	if err != nil {
		return fmt.Errorf(
			"open download source: %w",
			err,
		)
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(
		destination,
		os.O_CREATE|
			os.O_WRONLY|
			os.O_TRUNC,
		0o600,
	)
	if err != nil {
		return fmt.Errorf(
			"create download destination: %w",
			err,
		)
	}

	_, copyErr := io.Copy(
		destinationFile,
		sourceFile,
	)

	closeErr :=
		destinationFile.Close()

	if copyErr != nil {
		_ = os.Remove(
			destination,
		)

		return fmt.Errorf(
			"copy download file: %w",
			copyErr,
		)
	}

	if closeErr != nil {
		_ = os.Remove(
			destination,
		)

		return fmt.Errorf(
			"close download file: %w",
			closeErr,
		)
	}

	if err := os.Remove(
		source,
	); err != nil {
		_ = os.Remove(
			destination,
		)

		return fmt.Errorf(
			"remove download source: %w",
			err,
		)
	}

	return nil
}
