package stdout

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/lestrrat-go/strftime"
)

const defaultFilenameTimeFormat = "%Y-%m-%dT%H-%M-%S"

type rotatingWriter struct {
	filenamePattern    string // strftime パターン (設定値そのまま)
	filename           string // 現在開いているファイルパス (展開済み)
	maxSize            int64
	maxAge             int
	maxBackups         int
	localTime          bool
	compress           bool
	compressType       CompressType
	compressWorkers    int
	filenameTimeFormat string

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
	timeFormat := cfg.FilenameTimeFormat
	if timeFormat == "" {
		timeFormat = defaultFilenameTimeFormat
	}
	maxSizeMB := cfg.MaxSize
	if maxSizeMB == 0 {
		maxSizeMB = 100 // default 100MB
	}
	rw := &rotatingWriter{
		filenamePattern:    cfg.Filename,
		maxSize:            int64(maxSizeMB) * 1024 * 1024,
		maxAge:             cfg.MaxAge,
		maxBackups:         cfg.MaxBackups,
		localTime:          cfg.LocalTime,
		compress:           cfg.Compress,
		compressType:       cfg.CompressType,
		compressWorkers:    workers,
		filenameTimeFormat: timeFormat,
		compressCh:         make(chan string, 1000),
	}
	if err := rw.openFile(rw.evaluateFilename(rw.now())); err != nil {
		return nil, err
	}
	for i := 0; i < workers; i++ {
		rw.wg.Add(1)
		go rw.compressWorker()
	}
	return rw, nil
}

func (rw *rotatingWriter) now() time.Time {
	t := time.Now()
	if !rw.localTime {
		return t.UTC()
	}
	return t
}

// evaluateFilename は filenamePattern を時刻で展開する。
// パターンに strftime ディレクティブが含まれない場合はそのまま返す。
func (rw *rotatingWriter) evaluateFilename(t time.Time) string {
	name, err := strftime.Format(rw.filenamePattern, t)
	if err != nil {
		return rw.filenamePattern
	}
	return name
}

// openFile は指定パスのファイルを開き、rw.filename を更新する。
func (rw *rotatingWriter) openFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	rw.file = f
	rw.filename = path
	rw.currentSize = info.Size()
	return nil
}

func (rw *rotatingWriter) Write(p []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// 時刻ベースローテーション: パターン展開結果が変わったら新ファイルへ切り替え
	newFilename := rw.evaluateFilename(rw.now())
	if newFilename != rw.filename {
		if err := rw.rotateByTime(newFilename); err != nil {
			return 0, err
		}
	}

	// サイズベースローテーション
	if rw.currentSize+int64(len(p)) > rw.maxSize {
		if err := rw.rotateBySize(); err != nil {
			return 0, err
		}
	}

	n, err := rw.file.Write(p)
	rw.currentSize += int64(n)
	return n, err
}

// rotateByTime は時刻境界でのローテーション。
// 現在ファイルは既に日時を含む名前なのでリネーム不要。新ファイルを開く。
func (rw *rotatingWriter) rotateByTime(newFilename string) error {
	oldFilename := rw.filename
	if rw.file != nil {
		if err := rw.file.Close(); err != nil {
			return err
		}
		rw.file = nil
	}
	if rw.compress {
		select {
		case rw.compressCh <- oldFilename:
		default:
		}
	}
	go rw.cleanup()
	return rw.openFile(newFilename)
}

// rotateBySize はサイズ超過でのローテーション。現在ファイルをタイムスタンプ付きにリネームする。
func (rw *rotatingWriter) rotateBySize() error {
	oldFilename := rw.filename
	if rw.file != nil {
		if err := rw.file.Close(); err != nil {
			return err
		}
		rw.file = nil
	}
	rotatedPath := rw.rotatedFilename(oldFilename)
	if err := os.Rename(oldFilename, rotatedPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if rw.compress {
		select {
		case rw.compressCh <- rotatedPath:
		default:
		}
	}
	go rw.cleanup()
	// サイズローテーション後は同じパターンで新ファイルを開く
	return rw.openFile(rw.evaluateFilename(rw.now()))
}

// rotatedFilename はサイズローテーション時のリネーム先パスを返す。
func (rw *rotatingWriter) rotatedFilename(current string) string {
	ext := filepath.Ext(current)
	prefix := strings.TrimSuffix(current, ext)
	suffix, _ := strftime.Format(rw.filenameTimeFormat, rw.now())
	return prefix + "-" + suffix + ext
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

// filenameStaticPrefix は strftime パターンの静的プレフィックス部分を返す。
// 例: "/var/log/cor.%Y%m%d" → "/var/log/cor."
func filenameStaticPrefix(pattern string) string {
	if idx := strings.IndexByte(pattern, '%'); idx >= 0 {
		return pattern[:idx]
	}
	return pattern
}

// cleanup は MaxAge と MaxBackups に基づいて古いバックアップファイルを削除する。
func (rw *rotatingWriter) cleanup() {
	staticPrefix := filenameStaticPrefix(rw.filenamePattern)
	dir := filepath.Dir(staticPrefix)
	basePrefix := filepath.Base(staticPrefix)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var backups []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// 現在アクティブなファイルは対象外
		if filepath.Join(dir, e.Name()) == rw.filename {
			continue
		}
		if strings.HasPrefix(e.Name(), basePrefix) {
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
