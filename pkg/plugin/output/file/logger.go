package stdout

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

type rotatingWriter struct {
	filename        string
	maxSize         int64
	maxAge          int
	maxBackups      int
	localTime       bool
	compress        bool
	compressType    CompressType
	compressWorkers int

	mu          sync.Mutex
	file        *os.File
	currentSize int64

	compressCh chan string
	wg         sync.WaitGroup
}

func newRotatingWriter(cfg *Logger) (*rotatingWriter, error) {
	workers := cfg.CompressWorkers
	if workers <= 0 {
		workers = 1
	}
	rw := &rotatingWriter{
		filename:        cfg.Filename,
		maxSize:         int64(cfg.MaxSize) * 1024 * 1024,
		maxAge:          cfg.MaxAge,
		maxBackups:      cfg.MaxBackups,
		localTime:       cfg.LocalTime,
		compress:        cfg.Compress,
		compressType:    cfg.CompressType,
		compressWorkers: workers,
		compressCh:      make(chan string, 1000),
	}
	if rw.maxSize == 0 {
		rw.maxSize = 100 * 1024 * 1024 // default 100MB
	}
	if err := rw.openOrCreate(); err != nil {
		return nil, err
	}
	for i := 0; i < workers; i++ {
		rw.wg.Add(1)
		go rw.compressWorker()
	}
	return rw, nil
}

func (rw *rotatingWriter) openOrCreate() error {
	if err := os.MkdirAll(filepath.Dir(rw.filename), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(rw.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	rw.file = f
	rw.currentSize = info.Size()
	return nil
}

func (rw *rotatingWriter) Write(p []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.currentSize+int64(len(p)) > rw.maxSize {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := rw.file.Write(p)
	rw.currentSize += int64(n)
	return n, err
}

func (rw *rotatingWriter) rotate() error {
	if rw.file != nil {
		if err := rw.file.Close(); err != nil {
			return err
		}
		rw.file = nil
	}
	rotatedPath := rw.rotatedFilename()
	if err := os.Rename(rw.filename, rotatedPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if rw.compress {
		select {
		case rw.compressCh <- rotatedPath:
		default:
			// チャネルが満杯の場合は圧縮をスキップし、書き込みをブロックしない
		}
	}
	go rw.cleanup()
	return rw.openOrCreate()
}

func (rw *rotatingWriter) rotatedFilename() string {
	ext := filepath.Ext(rw.filename)
	prefix := strings.TrimSuffix(rw.filename, ext)
	t := time.Now()
	if !rw.localTime {
		t = t.UTC()
	}
	return fmt.Sprintf("%s-%s%s", prefix, t.Format("2006-01-02T15-04-05.000"), ext)
}

// compressWorker はチャネルからローテーション済みファイルを受け取り非同期で圧縮する。
func (rw *rotatingWriter) compressWorker() {
	defer rw.wg.Done()
	for path := range rw.compressCh {
		switch rw.compressType {
		case CompressTypeGzip:
			compressGzip(path)
		default: // zstd (default)
			compressZstd(path)
		}
	}
}

func compressGzip(path string) {
	src, err := os.Open(path)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.Create(path + ".gz")
	if err != nil {
		return
	}

	w := gzip.NewWriter(dst)
	if _, err := io.Copy(w, src); err != nil {
		w.Close()
		dst.Close()
		os.Remove(path + ".gz")
		return
	}
	if err := w.Close(); err != nil {
		dst.Close()
		os.Remove(path + ".gz")
		return
	}
	dst.Close()
	src.Close()
	os.Remove(path)
}

func compressZstd(path string) {
	src, err := os.Open(path)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.Create(path + ".zst")
	if err != nil {
		return
	}

	w, err := zstd.NewWriter(dst)
	if err != nil {
		dst.Close()
		os.Remove(path + ".zst")
		return
	}
	if _, err := io.Copy(w, src); err != nil {
		w.Close()
		dst.Close()
		os.Remove(path + ".zst")
		return
	}
	if err := w.Close(); err != nil {
		dst.Close()
		os.Remove(path + ".zst")
		return
	}
	dst.Close()
	src.Close()
	os.Remove(path)
}

// cleanup は MaxAge と MaxBackups に基づいて古いバックアップファイルを削除する。
func (rw *rotatingWriter) cleanup() {
	dir := filepath.Dir(rw.filename)
	base := filepath.Base(rw.filename)
	ext := filepath.Ext(base)
	prefix := strings.TrimSuffix(base, ext) + "-"

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var backups []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), prefix) {
			backups = append(backups, e)
		}
	}
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() < backups[j].Name()
	})

	if rw.maxAge > 0 {
		cutoff := time.Now().Add(-time.Duration(rw.maxAge) * 24 * time.Hour)
		var remaining []os.DirEntry
		for _, e := range backups {
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(dir, e.Name()))
			} else {
				remaining = append(remaining, e)
			}
		}
		backups = remaining
	}

	if rw.maxBackups > 0 && len(backups) > rw.maxBackups {
		for _, e := range backups[:len(backups)-rw.maxBackups] {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

func (rw *rotatingWriter) Close() error {
	close(rw.compressCh)
	rw.wg.Wait()
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.file != nil {
		err := rw.file.Close()
		rw.file = nil
		return err
	}
	return nil
}
