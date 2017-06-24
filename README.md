picadastra [![Build Status](https://travis-ci.org/fawick/picadastra.svg?branch=master)](https://travis-ci.org/fawick/picadastra)
=========

A simple photo importing tool that automatically sorts the photos into date-based folders.

Installation
------------

You need Go 1.4 or higher installed. A `go get` command is sufficient to install picadastra.

	$ go get github.com/fawick/picadastra

Usage
-----

The command below assumes that picadastra can be found in your `PATH` environment variable. 

	$ picadastra /media/SDCARD ~/Pictures
	
	42 new files, 0 overwritten,  skipped, 0 merged

`picadastra` will create subdirectories in `~/Pictures` that are named after
the creation date of the photo (read from EXIF tags). The default date format
is `yyyy-mm-dd`, e.g. `2015-03-14`. A different date format string can be
provided with the parameter `-d <format>`. The syntax of `<format`> is the same
as for Go's [func (time.Time) Format](http://golang.org/pkg/time/#Time.Format).
Example:

	$ picadastra -d 20060102_15 /media/SDCARD ~/Pictures


