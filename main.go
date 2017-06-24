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

	"github.com/cheggaaa/pb"
	"github.com/rwcarlsen/goexif/exif"
)

const (
	defaultDateFormat = "2006-01-02"
)

type TransferTask struct {
	srcDir       string
	dstDir       string
	dateFormat   string
	verbose      bool
	superVerbose bool
	force        bool
	skipVideos   bool
	statistics   struct {
		created     int
		overwritten int
		skipped     int
		merged      int
	}
}

func (tt *TransferTask) printStatistics() {
	fmt.Printf("\n%d new files, %d overwritten, %d skipped, %d merged\n",
		tt.statistics.created,
		tt.statistics.overwritten,
		tt.statistics.skipped,
		tt.statistics.merged,
	)
}

func main() {
	var (
		verboseFlag      = flag.Bool("v", false, "Print verbose output")
		superVerboseFlag = flag.Bool("vv", false, "Even report identical files")
		dateFormatFlag   = flag.String("d", defaultDateFormat, "Date Format for directory names")
		forceFlag        = flag.Bool("f", false, "Overwrite existing files when sizes do not match")
		skipVideosFlag   = flag.Bool("s", false, "Skip Video files")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: picadastra [options] <source dir> <destination dir>\n\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}
	flag.Parse()
	tt := TransferTask{
		dateFormat:   *dateFormatFlag,
		verbose:      *verboseFlag || *superVerboseFlag,
		superVerbose: *verboseFlag,
		force:        *forceFlag,
		skipVideos:   *skipVideosFlag,
	}
	args := flag.Args()
	switch len(args) {
	default:
		flag.Usage()
		os.Exit(1)
	case 1:
		tt.srcDir = args[0]
		u, _ := user.Current()
		tt.dstDir = filepath.Join(u.HomeDir, "Pictures")
	case 2:
		tt.srcDir = args[0]
		tt.dstDir = args[1]
	}

	if err := tt.Exec(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (tt *TransferTask) Exec() error {
	defer tt.printStatistics()
	si, err := os.Stat(tt.srcDir)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("invalid source: %s is not a directory", tt.srcDir)
	}
	di, err := os.Stat(tt.srcDir)
	if err != nil {
		return err
	}
	if !di.IsDir() {
		return fmt.Errorf("invalid destination: %s is not a directory", tt.dstDir)
	}
	return filepath.Walk(tt.srcDir, tt.walkPhotoVideos)
}

type CameraItem struct {
	Path    string
	ModTime time.Time
	Size    int64
}

func (tt *TransferTask) walkPhotoVideos(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	l := strings.ToLower(info.Name())
	if strings.HasSuffix(l, ".jpg") || strings.HasSuffix(l, "*.jpeg") {
		return tt.importItem(CameraItem{Path: path, ModTime: info.ModTime(), Size: info.Size()})
	} else if strings.HasSuffix(l, ".mov") || strings.HasSuffix(l, ".mp4") {
		if tt.skipVideos {
			fmt.Printf("Ignoring video file %s\n", path)
			return nil
		}
		return tt.importItem(CameraItem{Path: path, ModTime: info.ModTime(), Size: info.Size()})
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
	si, err := os.Stat(from)
	if err != nil {
		return err
	}
	bar := pb.New(int(si.Size())).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10) // TODO increase refreshrate interval to 100ms
	bar.SetWidth(78).SetMaxWidth(78)
	bar.ShowSpeed = true
	bar.ShowPercent = false
	bar.Start()
	defer bar.Finish()
	writer := io.MultiWriter(d, bar)
	if _, err := io.Copy(writer, s); err != nil {
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
func merge(from, to string, verbose bool) error {
	r, err := exec.LookPath("rsync")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: rsync not found. Falling back to simple copy")
		return cp(from, to)
	}
	c := exec.Command(r, "-tP", "--inplace", from, to)
	if verbose {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}
	return c.Run()
}

func datePath(path string, format string) (string, error) {
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
	return tm.Format(format), nil
}

func (tt *TransferTask) importItem(ci CameraItem) error {
	dp, err := datePath(ci.Path, tt.dateFormat)
	if err != nil {
		return fmt.Errorf("%s: %v", ci.Path, err)
	}
	d := filepath.Join(tt.dstDir, dp, filepath.Base(ci.Path))
	di, err := os.Stat(d)
	if os.IsNotExist(err) {
		if tt.verbose {
			fmt.Println("Copying new file:", ci.Path, "==>", d)
		}
		if err = cp(ci.Path, d); err != nil {
			return fmt.Errorf("Cannot copy %s: %v", ci.Path, err)
		}
		tt.statistics.created++
		return nil
	} else if err != nil {
		return fmt.Errorf("Cannot stat %s: %v", d, err)
	}
	if ci.Size != di.Size() {
		if tt.force {
			if ci.Size > 10*1024*1024 {
				if tt.verbose {
					fmt.Println("Merging:", ci.Path, "==>", d)
				}
				if err = merge(ci.Path, d, tt.verbose); err != nil {
					return fmt.Errorf("Cannot merge %s: %v", ci.Path, err)
				}
				tt.statistics.merged++
				return nil
			} else {
				if tt.verbose {
					fmt.Println("Overwriting:", ci.Path, "==>", d)
				}
				if err = cp(ci.Path, d); err != nil {
					return fmt.Errorf("Cannot copy %s: %v", ci.Path, err)
				}
				tt.statistics.overwritten++
			}
		} else {
			if tt.verbose {
				fmt.Printf("Warning: Skipping %s ==> %s (%d bytes vs %d bytes)\n", ci.Path, d, ci.Size, di.Size())
			}
			tt.statistics.skipped++
		}
	} else if tt.superVerbose {
		fmt.Printf("Already identical %s == %s\n", ci.Path, d)
	}
	return nil
}
