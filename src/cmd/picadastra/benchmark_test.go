package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"
)

// goexif brings enough samples, no need to add our own
var sampleFile = "../../../vendor/src/github.com/rwcarlsen/goexif/exif/sample1.jpg"

func BenchmarkDatePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		datePath(sampleFile)
	}
}

// For the sake of comparison, will be skipped if there is now exiftool on the
// machine
func BenchmarkDatePathExifTool(b *testing.B) {
	et, err := exec.LookPath("exiftool")
	if err != nil {
		b.Skip("No exiftool available. No worries")
	}
	a, err := exec.LookPath("awk")
	if err != nil {
		b.Skip("No awk available. Odd, but no problem.")
	}
	s, err := exec.LookPath("sh")
	if err != nil {
		b.Skip("No shell available. You got to be kidding me.")
	}
	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		buf.WriteString(fmt.Sprintf("%s -DateTimeOriginal -d %%Y-%%m-%%d %s | %s {'printf $4'}\n", et, sampleFile, a))
		c := exec.Command(s)
		c.Stdin = buf
		o, _ := c.CombinedOutput()
		bytes.TrimSpace(o)
	}
}
