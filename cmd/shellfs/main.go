package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"bazil.org/fuse/fuseutil"
)

func usage() {
	fmt.Printf("Usage of %s:\n", os.Args[0])
	fmt.Printf("\t%s MOUNTPOINT FROM\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 2 {
		usage()
		os.Exit(1)
	}
	mountpoint := flag.Arg(0)
	source := flag.Arg(1)

	c, err := fuse.Mount(mountpoint,
		fuse.FSName("shell-command-fs"),
		fuse.Subtype("shellfs"))

	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	defer fuse.Unmount(mountpoint) // doesn't do anything...

	err = fs.Serve(c, ShellFileSystem{Origin: source})
	if err != nil {
		log.Fatal(err)
	}
}

// ShellFileSystem roots the shell command file system
type ShellFileSystem struct {
	Origin string
}

type Dir struct {
	Origin string
	Files  *map[string]File
}

type File struct {
	Attributes *fuse.Attr
	Inode      uint64
	Mode       os.FileMode
}

type FileHandle struct {
}

func (ShellFileSystem) Root() (fs.Node, error) {
	files := make(map[string]File)
	files["test-large-file"] = File{Attributes: &fuse.Attr{Inode: 2, Mode: 0o444}}
	return Dir{Origin: "/", Files: &files}, nil
}

func (Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0o555
	return nil
}

func (self Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	tmp, present := (*self.Files)[name]
	if present {
		return tmp, nil
	}
	return nil, fuse.ENOENT
}

func (Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return []fuse.Dirent{{Inode: 2, Name: "test-large-file", Type: fuse.DT_File}}, nil
}

func (my File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = my.Attributes.Inode
	a.Mode = my.Attributes.Mode
	a.Size = 345
	fmt.Printf("%#v\n", a)
	return nil
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	fmt.Printf("%#v\n", req)
	fmt.Printf("%#v\n", resp)
	resp.Flags |= fuse.OpenNonSeekable | fuse.OpenDirectIO
	return &FileHandle{}, nil
}

func (fh *FileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	fmt.Printf("%#v\n", req)
	fmt.Printf("%#v\n", resp)
	fuseutil.HandleRead(req, resp, []byte("Hello\n"))
	return nil
}

func (fh *FileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	fmt.Printf("%#v\n", req)
	return nil
}

// Confirm the various types are implement the necessary interfaces
var _ fs.HandleReader = &FileHandle{}
var _ fs.Node = (*File)(nil)
var _ fs.Node = (*Dir)(nil)
var _ fs.FS = (*ShellFileSystem)(nil)
var _ fs.NodeStringLookuper = (*Dir)(nil)
var _ fs.NodeOpener = (*File)(nil)
var _ fs.Handle = (*FileHandle)(nil)
var _ fs.HandleReleaser = (*FileHandle)(nil)
var _ fs.HandleReader = (*FileHandle)(nil)
