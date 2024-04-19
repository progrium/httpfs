package main

import (
	"context"
	"errors"
	"hash/fnv"
	"io"
	iofs "io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type fuseMount struct {
	fs   iofs.FS
	path string

	*fuse.Server
}

func (m *fuseMount) Prepare() error {
	exec.Command("umount", m.path).Run()

	if err := os.MkdirAll(m.path, 0755); err != nil {
		return errors.New("unable to mkdir")
	}

	return nil
}

func (m *fuseMount) Mount() (err error) {
	opts := &fs.Options{
		UID: uint32(os.Getuid()),
		GID: uint32(os.Getgid()),
	}
	opts.Debug = false

	m.Server, err = fs.Mount(m.path, &node{fs: m.fs}, opts)
	if err != nil {
		return err
	}

	return nil
}

func (m *fuseMount) Unmount() error {
	if m.Server == nil {
		exec.Command("umount", m.path).Run()
		return nil
	}
	return m.Server.Unmount()
}

func fakeIno(s string) uint64 {
	h := fnv.New64a() // FNV-1a 64-bit hash
	h.Write([]byte(s))
	return h.Sum64()
}

func applyFileInfo(out *fuse.Attr, fi iofs.FileInfo) {
	stat := fi.Sys()
	if s, ok := stat.(*syscall.Stat_t); ok {
		out.FromStat(s)
		return
	}
	out.Mtime = uint64(fi.ModTime().Unix())
	out.Mtimensec = uint32(fi.ModTime().UnixNano())
	out.Mode = uint32(fi.Mode())
	out.Size = uint64(fi.Size())
}

type node struct {
	fs.Inode
	fs   iofs.FS
	path string
}

var _ = (fs.NodeGetattrer)((*node)(nil))

func (n *node) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	// log.Println("getattr", n.path)

	fi, err := iofs.Stat(n.fs, ".")
	if err != nil {
		return sysErrno(err)
	}
	applyFileInfo(&out.Attr, fi)

	return 0
}

var _ = (fs.NodeReaddirer)((*node)(nil))

func (n *node) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	// log.Println("readdir", n.path)

	entries, err := iofs.ReadDir(n.fs, ".")
	if err != nil {
		return nil, sysErrno(err)
	}

	var fentries []fuse.DirEntry
	for _, entry := range entries {
		fentries = append(fentries, fuse.DirEntry{
			Name: entry.Name(),
			Mode: uint32(entry.Type()),
			Ino:  fakeIno(filepath.Join(n.path, entry.Name())),
		})
	}

	return fs.NewListDirStream(fentries), 0
}

var _ = (fs.NodeOpendirer)((*node)(nil))

func (r *node) Opendir(ctx context.Context) syscall.Errno {
	// log.Println("opendir", r.path)
	return 0
}

var _ = (fs.NodeLookuper)((*node)(nil))

func (n *node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	// log.Println("lookup", n.path, name)

	fi, err := iofs.Stat(n.fs, name)
	if err != nil {
		return nil, sysErrno(err)
	}

	applyFileInfo(&out.Attr, fi)

	subfs, err := iofs.Sub(n.fs, name)
	if err != nil {
		return nil, sysErrno(err)
	}

	mode := fuse.S_IFREG
	if fi.IsDir() {
		mode = fuse.S_IFDIR
	}

	return n.Inode.NewPersistentInode(ctx, &node{
		fs:   subfs,
		path: filepath.Join(n.path, name),
	}, fs.StableAttr{
		Mode: uint32(mode),
		Ino:  fakeIno(filepath.Join(n.path, name)),
	}), 0
}

var _ = (fs.NodeOpener)((*node)(nil))

func (n *node) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	// log.Println("open", n.path)

	f, err := n.fs.Open(".") // should be OpenFile
	if err != nil {
		return nil, 0, sysErrno(err)
	}

	// buffer entire contents to support ReaderAt style reading,
	// required for Direct I/O mode, which is needed to avoid caching
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, sysErrno(err)
	}

	return &handle{data: data, path: n.path}, fuse.FOPEN_DIRECT_IO, 0
}

type handle struct {
	data []byte
	path string
}

var _ = (fs.FileReader)((*handle)(nil))

func (h *handle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	// log.Println("read", h.path)

	end := off + int64(len(dest))
	if end > int64(len(h.data)) {
		end = int64(len(h.data))
	}

	return fuse.ReadResultData(h.data[off:end]), 0
}

func sysErrno(err error) syscall.Errno {
	log.Println("ERR:", err)
	switch err {
	case nil:
		return syscall.Errno(0)
	case os.ErrPermission:
		return syscall.EPERM
	case os.ErrExist:
		return syscall.EEXIST
	case os.ErrNotExist:
		return syscall.ENOENT
	case os.ErrInvalid:
		return syscall.EINVAL
	}

	switch t := err.(type) {
	case syscall.Errno:
		return t
	case *os.SyscallError:
		return t.Err.(syscall.Errno)
	case *os.PathError:
		return sysErrno(t.Err)
	case *os.LinkError:
		return sysErrno(t.Err)
	}
	log.Println("!! unsupported error type:", err)
	return syscall.EINVAL
}
