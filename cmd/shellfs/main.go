package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
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
	_ = source

	c, err := fuse.Mount(mountpoint,
		fuse.FSName("shell-command-fs"),
		fuse.Subtype("shellfs"))

	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	err = fs.Serve(c, ShellFileSystem{})
	if err != nil {
		log.Fatal(err)
	}
}

// ShellFileSystem roots the shell command file system
type ShellFileSystem struct{}

type Dir struct {
	origin string
	Files  *map[string]File
}

type File struct {
	Att *fuse.Attr
}

func (ShellFileSystem) Root() (fs.Node, error) {
	files := make(map[string]File)
	files["Test\\ Large\\ File"] = File{Att: &fuse.Attr{Inode: 2, Mode: 0o444}}
	return Dir{origin: "/", Files: &files}, nil
}

func (Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0o555
	return nil
}

func (self Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	files := *self.Files
	fmt.Printf("%+v\n", files)
	fmt.Printf("%s\n", name)
	return files[name], nil
}

func (Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return []fuse.Dirent{{Inode: 2, Name: "Test Large File", Type: fuse.DT_File}}, nil
}

func (my File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = my.Att.Inode
	a.Mode = my.Att.Mode
	return nil
}
