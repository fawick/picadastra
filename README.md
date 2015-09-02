picadastra
=========

A simple photo importing tool that automatically sorts the photos into date-based folders.

Installation
------------

You need Go 1.4 or higher installed.

`picadastra` uses [gb as its build tool](https://github.com/constabulary/gb). Install gb if you haven't already:

	$ go get github.com/constabulary/gb/...

Then, clone `picadastra` with your favorite cloning method, and run gb. e.g.

	$ git clone https://github.com/fawick/picadastra
	$ cd picadastra
	$ gb build

The compiled `picadastra` binary will be in the subdirectory `bin`.

Usage
-----

	$ bin/picadastra /media/SDCARD ~/Pictures
	
	42 new files, 0 overwritten,  skipped, 0 merged

`picadastra` will create subdirectories in `~/Pictures` that are named after
the creation date of the photo (read from EXIF tags). The default date format
is `yyyy-mm-dd`, e.g. `2015-03-14`. A different date format string can be
provided with the parameter `-d <format>`. The syntax of `<format`> is the same
as for Go's [func (time.Time) Format](http://golang.org/pkg/time/#Time.Format).
Example:

	$ bin/picadasrta -d 20060102_15 /media/SDCARD ~/Pictures


