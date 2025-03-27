package file

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dormoron/phantasm/config"
)

// Source 是基于文件的配置源
type Source struct {
	path   string
	format string
}

// NewSource 创建一个基于文件的配置源
func NewSource(path string) config.Source {
	return &Source{
		path:   path,
		format: detectFormat(path),
	}
}

// detectFormat 根据文件扩展名检测配置格式
func detectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if len(ext) > 0 {
		// 移除前导点
		ext = ext[1:]
	}
	switch ext {
	case "json":
		return "json"
	case "yaml", "yml":
		return "yaml"
	case "toml":
		return "toml"
	case "xml":
		return "xml"
	case "properties", "prop", "props":
		return "properties"
	default:
		return ""
	}
}

// Load 加载文件配置
func (s *Source) Load() ([]*config.KeyValue, error) {
	// 判断文件是否存在
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return nil, err
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	// 提取文件名作为顶级键
	fileName := filepath.Base(s.path)
	if ext := filepath.Ext(fileName); len(ext) > 0 {
		fileName = fileName[:len(fileName)-len(ext)]
	}

	return []*config.KeyValue{
		{
			Key:    fileName,
			Value:  string(data),
			Format: s.format,
		},
	}, nil
}

// Watch 监视配置文件变化
func (s *Source) Watch() (config.Watcher, error) {
	return newWatcher(s.path)
}

// Watcher 是文件配置监视器
type Watcher struct {
	path     string
	format   string
	closeCh  chan struct{}
	changeCh chan []*config.KeyValue
}

// newWatcher 创建一个新的文件监视器
func newWatcher(path string) (*Watcher, error) {
	w := &Watcher{
		path:     path,
		format:   detectFormat(path),
		closeCh:  make(chan struct{}),
		changeCh: make(chan []*config.KeyValue, 1),
	}

	return w, nil
}

// Next 阻塞直到接收到文件变化或上下文被取消
func (w *Watcher) Next() ([]*config.KeyValue, error) {
	select {
	case <-w.closeCh:
		return nil, config.ErrWatcherClosed
	case kv := <-w.changeCh:
		return kv, nil
	}
}

// Stop 停止监视
func (w *Watcher) Stop() error {
	close(w.closeCh)
	return nil
}
