package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"

	io_fs "io/fs"
	"os/exec"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	_ "bazil.org/fuse/fuseutil"
)

func usage() {
	fmt.Printf("Usage of %s:\n", os.Args[0])
	fmt.Printf("\t%s [-s /path/to/size/pipe] MOUNTPOINT FROM\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var blockingDonePipe = flag.String("s", "", "One line 'Sizes Synchronized' will be written to a pipe at this location. You can block on that pipe to wait for sizes to balance.")
	flag.Usage = usage
	flag.Parse()

	var err error

	if flag.NArg() != 2 {
		usage()
		os.Exit(1)
	}
	mountpoint := flag.Arg(0)
	source := flag.Arg(1)

	var fifo *os.File
	if *blockingDonePipe != "" {
		fifo, err = os.OpenFile(*blockingDonePipe, os.O_WRONLY, 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	c, err := fuse.Mount(mountpoint,
		fuse.FSName("shell-command-fs"),
		fuse.Subtype("shellfs"))

	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	defer fuse.Unmount(mountpoint) // ... never gets called

	fmt.Println("Starting scan...")
	stable := make(chan StableFS)
	entry_passage := make(chan Entry)
	lifecycle := make(chan Lifecycle)
	go assemble_entries(entry_passage, lifecycle, stable)

	go func() {
		if fifo != nil {
			select {
			case <-stable:
				fifo.Write([]byte("Sizes Synchronized\n"))
				fifo.Close()
			}
		}
	}()

	parse_origin_dir(source, entry_passage, lifecycle)

	fmt.Println("Serving files!")
	err = fs.Serve(c, ShellFileSystem{origin: source})
	fuse.Unmount(mountpoint)
	if err != nil {
		log.Fatal(err)
	}
}

var TREE map[string][]*Entry
var ENTRIES map[string]*Entry

type Lifecycle struct{}
type StableFS struct{}

func parse_origin_dir(path string, out chan<- Entry, lifecycle chan<- Lifecycle) error {
	err := filepath.WalkDir(path, parse_out_command_files(out))
	lifecycle <- Lifecycle{}
	return err
}

func match_command_file_name(name string) bool {
	return name[0] == '#' && name[len(name)-1] == '#'
}

func box_command_file_name(name string) string {
	return "#" + name + "#"
}

func unbox_command_file_name(name string) string {
	return name[1 : len(name)-1]
}

func parse_out_command_files(out chan<- Entry) func(string, io_fs.DirEntry, error) error {
	return func(path string, info io_fs.DirEntry, err error) error {
		is_dir := info.IsDir()
		true_path := filepath.Dir(path)
		name := info.Name()
		is_command_file := match_command_file_name(name) && is_dir
		final_name := name
		if is_command_file {
			final_name = unbox_command_file_name(name)
		}
		var ft EntryType
		if is_command_file {
			ft = ET_CommandFile
		} else if is_dir {
			ft = ET_Directory
		} else {
			ft = ET_ReadOnlyFile
		}
		out <- Entry{file_type: ft,
			path: true_path,
			name: final_name}
		if is_command_file {
			return io_fs.SkipDir
		}
		return nil
	}
}

func assemble_entries(in <-chan Entry, lifecycle <-chan Lifecycle, done chan<- StableFS) {
	sizes := make(chan Entry)
	var expect_sizes int64 = 0
	tree := make(map[string][]*Entry)
	entries := make(map[string]*Entry)
	var inode uint64 = 0
	for {
		select {
		case item := <-in:
			inode++
			item.inode = inode
			go item.compute_size(sizes)
			tree[item.path] = append(tree[item.path], &item)
			entries[filepath.Join(item.path, item.name)] = &item
		case <-lifecycle:
			TREE = tree
			ENTRIES = entries
			tree = make(map[string][]*Entry)
			entries = make(map[string]*Entry)
			fmt.Println("The tree:")
			for k, v := range TREE {
				fmt.Printf("%s:\n", k)
				for k, v := range v {
					fmt.Printf("\t%d: %#v\n", k, v)
				}
			}
			fmt.Println("Individual entries:")
			for k, v := range ENTRIES {
				fmt.Printf("%s: %#v\n", k, v)
			}
			expect_sizes += int64(inode)
			inode = 0
		case <-sizes:
			expect_sizes -= 1
			if expect_sizes == 0 {
				done <- StableFS{}
			}
		}
	}
}

func command_file_to_size(e Entry) uint64 {
	path := filepath.Join(e.path, box_command_file_name(e.name))
	cmd := exec.Cmd{Path: "./size", Dir: path}
	out, err := cmd.Output()
	if err == nil {
		// newline at the end can cause problems
		uint_str := strings.TrimSpace(string(out))
		size, err := strconv.ParseUint(uint_str, 10, 64)
		if err == nil {
			return size
		}
	} else { // that comamnd errored...
		// run the actual command and hope its size will always be the
		// same on subsequent runs
		bytes_read := uint64(0)
		data := make([]byte, 1<<22) // 4 meg
		cmd = exec.Cmd{Path: "./cmd", Dir: path}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			log.Print(err)
			return 0
		}
		for {
			n, err := stdout.Read(data)
			bytes_read = bytes_read + uint64(n)
			if err == io.ErrUnexpectedEOF || err == io.EOF {
				break
			} else if err != nil {
				return 0
			}
		}
		return bytes_read
	}
	return 0
}

func read_only_file_to_size(e Entry) uint64 {
	path := filepath.Join(e.path, e.name)
	info, err := os.Stat(path)
	if err != nil {
		log.Println(err)
		return 0
	}
	return uint64(info.Size())
}

func (e *Entry) compute_size(done chan<- Entry) {
	e.size = math.MaxInt64
	if e.file_type == ET_Directory {
		e.size = 4096
	} else if e.file_type == ET_CommandFile {
		e.size = command_file_to_size(*e)
	} else if e.file_type == ET_ReadOnlyFile {
		e.size = read_only_file_to_size(*e)
	}
	done <- *e
}

type EntryType int

const (
	ET_CommandFile EntryType = iota
	ET_Directory
	ET_ReadOnlyFile
)

// MakeEntry is used to help populate the filesystem
type Entry struct {
	file_type EntryType
	inode     uint64
	name      string
	path      string
	size      uint64
}

// ShellFileSystem roots the shell command file system
type ShellFileSystem struct {
	origin string
}

// Dir is a directory in the filesystem. It has a source location
type Dir struct {
	origin string
	size   uint64
	inode  uint64
}

// CommandFile is a file in the filesystem. It has a source location
type CommandFile struct {
	from  string
	size  uint64
	inode uint64
}

// ReadOnlyFile is a real file in the filesystem.
type ReadOnlyFile struct {
	from  string
	size  uint64
	inode uint64
}

// CommandFileHandle is an open file descriptor, with what's necessary to read.
type CommandFileHandle struct {
	stdout io.ReadCloser
	cmd    exec.Cmd
}

// ReadOnlyFileHandle is a real file in the filesystem.
type ReadOnlyFileHandle struct {
	r io.ReadCloser
}

//////////////////////////////////////////////////////////////////
// Root
/////////////////////////////////////////////////////////////////

func (sfs ShellFileSystem) Root() (fs.Node, error) {
	return Dir{origin: sfs.origin, size: 4096, inode: 1}, nil
}

//////////////////////////////////////////////////////////////////
// Dir
/////////////////////////////////////////////////////////////////

func (d Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	full_path := filepath.Join(d.origin, name)
	e := ENTRIES[full_path]
	if e == nil {
		return nil, fuse.ENOENT
	} else if e.file_type == ET_CommandFile {
		var rV CommandFile
		rV.size = e.size
		rV.from = filepath.Join(d.origin, box_command_file_name(name))
		rV.inode = e.inode
		return rV, nil
	} else if e.file_type == ET_Directory {
		var rV Dir
		rV.size = e.size
		rV.origin = full_path
		rV.inode = e.inode
		return rV, nil
	} else if e.file_type == ET_ReadOnlyFile {
		var rV ReadOnlyFile
		rV.inode = e.inode
		rV.from = filepath.Join(d.origin, name)
		rV.size = e.size
		return rV, nil
	}
	return nil, errors.New("Type not compatible")
}

func (d Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var rV []fuse.Dirent
	for _, e := range TREE[d.origin] {
		var de fuse.Dirent
		de.Inode = e.inode
		de.Name = e.name
		if e.file_type == ET_CommandFile || e.file_type == ET_ReadOnlyFile {
			de.Type = fuse.DT_File
		} else if e.file_type == ET_Directory {
			de.Type = fuse.DT_Dir
		}
		rV = append(rV, de)
	}
	return rV, nil
}

func (d Dir) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = d.inode
	attr.Size = d.size
	attr.Mode = os.ModeDir | 0o755
	return nil
}

//////////////////////////////////////////////////////////////////
// CommandFile
/////////////////////////////////////////////////////////////////

func (f CommandFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	cmd := exec.Cmd{Path: "./cmd", Dir: f.from}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Print(err)
		return nil, err
	}
	resp.Flags |= fuse.OpenNonSeekable
	return CommandFileHandle{cmd: cmd, stdout: stdout}, nil
}

func (f CommandFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = f.inode
	// EOF will be signaled when Read returns less than asked, so...
	attr.Size = f.size // This must be as large or larger than the target data
	attr.Mode = 0o644
	return nil
}

//////////////////////////////////////////////////////////////////
// CommandFileHandler
/////////////////////////////////////////////////////////////////

func (cfh CommandFileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	buf := make([]byte, req.Size)
	n, err := io.ReadFull(cfh.stdout, buf)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	resp.Data = buf[:n]
	return err
}

func (cfh CommandFileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	cfh.cmd.Process.Signal(syscall.SIGTERM)
	return nil
}

//////////////////////////////////////////////////////////////////
// ReadOnlyFile
/////////////////////////////////////////////////////////////////

func (rof ReadOnlyFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	file, err := os.Open(rof.from)
	if err != nil {
		return nil, err
	}
	resp.Flags |= fuse.OpenNonSeekable
	return ReadOnlyFileHandle{r: file}, nil
}

func (rof ReadOnlyFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = rof.inode
	// EOF will be signaled when Read returns less than asked, so...
	attr.Size = rof.size // This must be as large or larger than the target data
	attr.Mode = 0o644
	return nil
}

//////////////////////////////////////////////////////////////////
// ReadOnlyFileHandler
/////////////////////////////////////////////////////////////////

func (rofh ReadOnlyFileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	buf := make([]byte, req.Size)
	n, err := io.ReadFull(rofh.r, buf)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		err = nil
	}
	resp.Data = buf[:n]
	return err
}

func (rofh ReadOnlyFileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return rofh.r.Close()
}

// Confirm the various types are implement the necessary interfaces
var _ fs.FS = (*ShellFileSystem)(nil)
var _ fs.Node = (*Dir)(nil)
var _ fs.HandleReadDirAller = (*Dir)(nil)
var _ fs.NodeStringLookuper = (*Dir)(nil)
var _ fs.Node = (*CommandFile)(nil)
var _ fs.NodeOpener = (*CommandFile)(nil)
var _ fs.Handle = (*CommandFileHandle)(nil)
var _ fs.HandleReleaser = (*CommandFileHandle)(nil)
var _ fs.HandleReader = (*CommandFileHandle)(nil)
