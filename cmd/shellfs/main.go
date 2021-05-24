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
	_ "bazil.org/fuse/fuseutil"
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

	defer fuse.Unmount(mountpoint) // ... never gets called

	err = fs.Serve(c, ShellFileSystem{origin: source})
	fuse.Unmount(mountpoint)
	if err != nil {
		log.Fatal(err)
	}
}

// ShellFileSystem roots the shell command file system
type ShellFileSystem struct {
	origin string
}

// Dir is a directory in the filesystem. It has a source location
type Dir struct {
	origin string
}

// File is a file in the filesystem. It has a source location
type File struct {
	from string
}

type CommandFileHandle struct {
	from string
}

//////////////////////////////////////////////////////////////////
// Root
/////////////////////////////////////////////////////////////////

func (sfs ShellFileSystem) Root() (fs.Node, error) {
	fmt.Printf("Returning Root()\n")
	return Dir{origin: sfs.origin}, nil
}

//////////////////////////////////////////////////////////////////
// Dir
/////////////////////////////////////////////////////////////////

func (d Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	fmt.Printf("%s\n", name)
	return File{from: d.origin + "my-info"}, nil
}

func (d Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("ReadDirAll")
	return []fuse.Dirent{
		{Inode: 2, Name: "my-info", Type: fuse.DT_File},
	}, nil
}

func (d Dir) Attr(ctx context.Context, attr *fuse.Attr) error {
	fmt.Printf("%#v\n", attr)
	attr.Inode = 1
	attr.Mode = os.ModeDir | 0o755
	return nil
}

//////////////////////////////////////////////////////////////////
// File
/////////////////////////////////////////////////////////////////

func (f File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	fmt.Printf("%#v\n", req)
	fmt.Printf("%#v\n", resp)
	return CommandFileHandle{from: f.from}, nil
}

func (f File) Attr(ctx context.Context, attr *fuse.Attr) error {
	fmt.Printf("%#v\n", attr)
	// EOF will be signaled when Read returns less than asked, so...
	attr.Size = 3 // This must be as large or larger than the target data
	attr.Mode = 0o644
	return nil
}

//////////////////////////////////////////////////////////////////
// CommandFileHandler
/////////////////////////////////////////////////////////////////

func (cfh CommandFileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	fmt.Printf("%#v\n", req)
	fmt.Printf("%#v\n", resp)
	resp.Data = []byte("Hi\n")
	return nil
}

func (cfh CommandFileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	fmt.Printf("%#v\n", req)
	return nil
}

// Confirm the various types are implement the necessary interfaces
var _ fs.FS = (*ShellFileSystem)(nil)
var _ fs.Node = (*Dir)(nil)
var _ fs.HandleReadDirAller = (*Dir)(nil)
var _ fs.NodeStringLookuper = (*Dir)(nil)
var _ fs.Node = (*File)(nil)
var _ fs.NodeOpener = (*File)(nil)
var _ fs.Handle = (*CommandFileHandle)(nil)
var _ fs.HandleReleaser = (*CommandFileHandle)(nil)
var _ fs.HandleReader = (*CommandFileHandle)(nil)
