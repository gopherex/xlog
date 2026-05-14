package sink

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
)

type File struct {
	mu     sync.Mutex
	file   *os.File
	path   string
	size   int64
	config FileConfig
}

type FileConfig struct {
	MaxSize    int64
	MaxBackups int
	Compress   bool
	Mode       os.FileMode
	Clock      func() time.Time
}

type FileOption func(*FileConfig)

const defaultMaxFileSize = 100 * 1024 * 1024

func DefaultFileConfig() FileConfig {
	return FileConfig{
		MaxSize:    defaultMaxFileSize,
		MaxBackups: 7,
		Mode:       0o644,
		Clock:      time.Now,
	}
}

func WithMaxSize(bytes int64) FileOption {
	return func(c *FileConfig) {
		c.MaxSize = bytes
	}
}

func WithMaxBackups(count int) FileOption {
	return func(c *FileConfig) {
		c.MaxBackups = count
	}
}

func WithCompress(enabled bool) FileOption {
	return func(c *FileConfig) {
		c.Compress = enabled
	}
}

func WithFileMode(mode os.FileMode) FileOption {
	return func(c *FileConfig) {
		c.Mode = mode
	}
}

func WithClock(clock func() time.Time) FileOption {
	return func(c *FileConfig) {
		c.Clock = clock
	}
}

func OpenFile(path string, opts ...FileOption) (*File, error) {
	config := DefaultFileConfig()
	for _, opt := range opts {
		opt(&config)
	}
	if config.Mode == 0 {
		config.Mode = 0o644
	}
	if config.Clock == nil {
		config.Clock = time.Now
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	file, size, err := openAppend(path, config.Mode)
	if err != nil {
		return nil, err
	}
	return &File{
		file:   file,
		path:   path,
		size:   size,
		config: config,
	}, nil
}

func (f *File) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file == nil {
		if err := f.open(); err != nil {
			return 0, err
		}
	}
	if f.shouldRotate(len(p)) {
		if err := f.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := f.file.Write(p)
	f.size += int64(n)
	return n, err
}

func (f *File) Sync() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.file == nil {
		return nil
	}
	return f.file.Sync()
}

func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.close()
}

func (f *File) open() error {
	file, size, err := openAppend(f.path, f.config.Mode)
	if err != nil {
		return err
	}
	f.file = file
	f.size = size
	return nil
}

func (f *File) close() error {
	if f.file == nil {
		return nil
	}
	err := f.file.Close()
	f.file = nil
	return err
}

func (f *File) shouldRotate(writeLen int) bool {
	if f.config.MaxSize <= 0 || f.size == 0 {
		return false
	}
	return f.size+int64(writeLen) > f.config.MaxSize
}

func (f *File) rotate() error {
	if err := f.close(); err != nil {
		return err
	}

	backup := f.backupName()
	if err := os.Rename(f.path, backup); err != nil && !os.IsNotExist(err) {
		return err
	}
	if f.config.Compress {
		compressed, err := compressFile(backup)
		if err != nil {
			return err
		}
		backup = compressed
	}
	if err := f.removeOldBackups(); err != nil {
		return err
	}
	return f.open()
}

func (f *File) backupName() string {
	dir := filepath.Dir(f.path)
	ext := filepath.Ext(f.path)
	base := strings.TrimSuffix(filepath.Base(f.path), ext)
	ts := f.config.Clock().UTC().Format("20060102T150405.000000000")
	name := filepath.Join(dir, fmt.Sprintf("%s-%s%s", base, ts, ext))
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return name
	}
	for i := 1; ; i++ {
		next := filepath.Join(dir, fmt.Sprintf("%s-%s.%03d%s", base, ts, i, ext))
		if _, err := os.Stat(next); os.IsNotExist(err) {
			return next
		}
	}
}

func (f *File) removeOldBackups() error {
	if f.config.MaxBackups < 0 {
		return nil
	}

	backups, err := f.backups()
	if err != nil {
		return err
	}
	removeCount := len(backups) - f.config.MaxBackups
	for i := 0; i < removeCount; i++ {
		if err := os.Remove(backups[i]); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (f *File) backups() ([]string, error) {
	dir := filepath.Dir(f.path)
	ext := filepath.Ext(f.path)
	base := strings.TrimSuffix(filepath.Base(f.path), ext)
	pattern := filepath.Join(dir, base+"-*"+ext)

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	gzFiles, err := filepath.Glob(pattern + ".gz")
	if err != nil {
		return nil, err
	}
	files = append(files, gzFiles...)
	sort.Strings(files)
	return files, nil
}

func openAppend(path string, mode os.FileMode) (*os.File, int64, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, mode)
	if err != nil {
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, err
	}
	return file, info.Size(), nil
}

func compressFile(path string) (string, error) {
	in, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer in.Close()

	outPath := path + ".gz"
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}

	gz := gzip.NewWriter(out)
	_, copyErr := io.Copy(gz, in)
	closeGzErr := gz.Close()
	closeOutErr := out.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeGzErr != nil {
		return "", closeGzErr
	}
	if closeOutErr != nil {
		return "", closeOutErr
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return outPath, nil
}
