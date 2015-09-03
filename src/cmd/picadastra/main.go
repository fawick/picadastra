package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

var (
	statistics struct {
		created     int
		overwritten int
		skipped     int
		merged      int
	}
)

var (
	srcDir       string
	dstDir       string
	verbose      = flag.Bool("v", false, "Print verbose output")
	superVerbose = flag.Bool("vv", false, "Even report identical files")
	dateFormat   = flag.String("d", "2006-01-02", "Date Format for directory names")
	forced       = flag.Bool("f", false, "Overwrite existing files when sizes do not match")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: picadastra [options] <source dir> <destination dir>\n\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}
}

func printStatistics() {
	fmt.Printf("\n%d new files, %d overwritten, %d skipped, %d merged\n",
		statistics.created,
		statistics.overwritten,
		statistics.skipped,
		statistics.merged)
}

func main() {
	flag.Parse()
	if *superVerbose {
		*verbose = true
	}
	args := flag.Args()
	if len(args) == 1 {
		srcDir = args[0]
		u, _ := user.Current()
		dstDir = filepath.Join(u.HomeDir, "Pictures")
	} else if len(args) == 2 {
		srcDir = args[0]
		dstDir = args[0]
	} else {
		flag.Usage()
		os.Exit(1)
	}
	si, err := os.Stat(srcDir)
	if err == nil && !si.IsDir() {
		err = fmt.Errorf("Source incorrect: %s is not a directory", srcDir)
	}
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	dstDir = args[1]
	di, err := os.Stat(srcDir)
	if err == nil && !di.IsDir() {
		err = fmt.Errorf("Destination incorrect: %s is not a directory", dstDir)
	}
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	err = filepath.Walk(srcDir, walkPhotoVideos)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	printStatistics()
}

type CameraItem struct {
	Path    string
	ModTime time.Time
	Size    int64
}

func walkPhotoVideos(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	l := strings.ToLower(info.Name())
	if strings.HasSuffix(l, ".jpg") || strings.HasSuffix(l, "*.jpeg") {
		return importPhoto(CameraItem{Path: path, ModTime: info.ModTime(), Size: info.Size()})
	} else if strings.HasSuffix(l, ".mov") {
		return importVideo(CameraItem{Path: path, ModTime: info.ModTime(), Size: info.Size()})
	}
	return nil
}

// cp copies the file from into to and synchronizes the atime and mtime to be rsync-compatible.
func cp(from, to string) error {
	s, err := os.Open(from)
	if err != nil {
		return err
	}
	defer s.Close()
	p := filepath.Dir(to)
	if err = os.MkdirAll(p, 0755); err != nil {
		return err
	}
	d, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	if err = d.Close(); err != nil {
		return err
	}
	i, err := os.Stat(from)
	if err != nil {
		return err
	}
	return os.Chtimes(to, i.ModTime(), i.ModTime())
}

// merge calls rsync for delta-copying the file. As merge will only be called
// for large files it should be okay to spawn a subprocess.
func merge(from, to string) error {
	r, err := exec.LookPath("rsync")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: rsync not found. Falling back to simple copy")
		return cp(from, to)
	}
	c := exec.Command(r, "-tP", "--inplace", from, to)
	if *verbose {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}
	return c.Run()
}

func datePath(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	x, err := exif.Decode(f)
	if err != nil {
		return "", err
	}
	tm, err := x.DateTime()
	if err != nil {
		return "", err
	}
	return tm.Format(*dateFormat), nil
}

func importItem(ci CameraItem) error {
	dp, err := datePath(ci.Path)
	if err != nil {
		return fmt.Errorf("%s: %v", ci.Path, err)
	}
	d := filepath.Join(dstDir, dp, filepath.Base(ci.Path))
	di, err := os.Stat(d)
	if os.IsNotExist(err) {
		if *verbose {
			fmt.Println("Copying new file:", ci.Path, "==>", d)
		}
		if err = cp(ci.Path, d); err != nil {
			return fmt.Errorf("Cannot copy %s: %v", ci.Path, err)
		}
		statistics.created++
		return nil
	} else if err != nil {
		return fmt.Errorf("Cannot stat %s: %v", d, err)
	}
	if ci.Size != di.Size() {
		if *forced {
			if ci.Size > 10*1024*1024 {
				if *verbose {
					fmt.Println("Merging:", ci.Path, "==>", d)
				}
				if err = merge(ci.Path, d); err != nil {
					return fmt.Errorf("Cannot merge %s: %v", ci.Path, err)
				}
				statistics.merged++
				return nil
			} else {
				if *verbose {
					fmt.Println("Overwriting:", ci.Path, "==>", d)
				}
				if err = cp(ci.Path, d); err != nil {
					return fmt.Errorf("Cannot copy %s: %v", ci.Path, err)
				}
				statistics.overwritten++
			}
		} else {
			if *verbose {
				fmt.Printf("Warning: Skipping %s ==> %s (%d bytes vs %d bytes)\n", ci.Path, d, ci.Size, di.Size())
			}
			statistics.skipped++
		}
	} else if *superVerbose {
		fmt.Printf("Already identical %s == %s\n", ci.Path, d)
	}
	return nil
}

func importPhoto(ci CameraItem) error {
	return importItem(ci)
}

func importVideo(ci CameraItem) error {
	return importItem(ci)
	return nil
}
