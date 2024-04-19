package main

import (
	"encoding/json"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var startTime time.Time

func init() {
	startTime = time.Now()
}

type httpFS_v1 struct {
	Dir []string `json:"dir,omitempty"`
}

type statCache struct {
	info
	lastTime time.Time
}

type httpFS struct {
	baseURL   string
	statCache map[string]statCache
}

func (fsys *httpFS) url(name string) string {
	if name == "." {
		name = ""
	}
	return strings.Join([]string{strings.TrimRight(fsys.baseURL, "/"), strings.TrimLeft(name, "/")}, "/")
}

func (fsys *httpFS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	return fsys.Open(name)
}

func (fsys *httpFS) Open(name string) (fs.File, error) {
	resp, err := http.DefaultClient.Get(fsys.url(name))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, fs.ErrNotExist
	}
	return &file{
		ReadCloser: resp.Body,
		Name:       name,
		FS:         fsys,
	}, nil
}

func (fsys *httpFS) stat(name string) (*info, error) {
	if fsys.statCache == nil {
		fsys.statCache = make(map[string]statCache)
	}
	fi, found := fsys.statCache[name]

	if found && time.Since(fi.lastTime).Milliseconds() < 1000 {
		return &fi.info, nil
	}

	resp, err := http.DefaultClient.Head(fsys.url(name))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, fs.ErrNotExist
	}

	// defaults
	i := &info{
		name:    filepath.Base(name),
		size:    0,
		mode:    0644,
		modTime: startTime.Unix(),
		isDir:   false,
	}

	disp := resp.Header.Get("Content-Disposition")
	if disp != "" {
		_, params, _ := mime.ParseMediaType(disp)
		if params != nil {
			i.name = params["filename"]
		}
	}

	length := resp.Header.Get("Content-Length")
	if length != "" {
		l, err := strconv.Atoi(length)
		if err == nil {
			i.size = int64(l)
		}
	}

	modTime := resp.Header.Get("Last-Modified")
	if modTime != "" {
		t, err := time.Parse(time.RFC1123, modTime)
		if err == nil {
			i.modTime = int64(t.Unix())
		}
	}

	typ := resp.Header.Get("Content-Type")
	if typ == "application/vnd.httpfs.v1+json" {
		i.isDir = true
		i.mode = 0755
	}

	mode := resp.Header.Get("Content-Permissions") // custom!
	if mode != "" {
		m, err := strconv.ParseUint(mode, 8, 32)
		if err == nil {
			i.mode = uint(m)
		}
	}

	fsys.statCache[name] = statCache{
		info:     *i,
		lastTime: time.Now(),
	}
	return i, nil
}

func (fsys *httpFS) Stat(name string) (fs.FileInfo, error) {
	return fsys.stat(name)
}

func (fsys *httpFS) ReadDir(name string) ([]fs.DirEntry, error) {
	resp, err := http.DefaultClient.Get(fsys.url(name))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, fs.ErrNotExist
	}

	// todo: check for application/vnd.httpfs.v1+json

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	var dirInfo httpFS_v1
	if err := json.Unmarshal(b, &dirInfo); err != nil {
		return nil, err
	}

	var out []fs.DirEntry
	for _, sub := range dirInfo.Dir {
		info, err := fsys.stat(filepath.Join(name, sub))
		if err != nil {
			return nil, err
		}
		out = append(out, info)
	}
	return out, nil
}

type file struct {
	io.ReadCloser
	Name string
	FS   *httpFS
}

func (f *file) Stat() (fs.FileInfo, error) {
	return f.FS.Stat(f.Name)
}

func (f *file) ReadDir(n int) ([]fs.DirEntry, error) {
	return f.FS.ReadDir(f.Name)
}

type info struct {
	name    string
	size    int64
	mode    uint
	modTime int64
	isDir   bool
}

func (i *info) Name() string       { return i.name }
func (i *info) Size() int64        { return i.size }
func (i *info) ModTime() time.Time { return time.Unix(i.modTime, 0) }
func (i *info) IsDir() bool        { return i.isDir }
func (i *info) Sys() any           { return nil }
func (i *info) Mode() fs.FileMode {
	if i.IsDir() {
		return fs.FileMode(i.mode) | fs.ModeDir
	}
	return fs.FileMode(i.mode)
}

// these allow it to act as DirInfo as well
func (i *info) Info() (fs.FileInfo, error) {
	return i, nil
}
func (i *info) Type() fs.FileMode {
	return i.Mode()
}
