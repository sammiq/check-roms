check-roms: a simple rom auditing tool in Go
============================================

This tool uses logiqx xml format dat files, as provided by your friendly preservation site, for verifying your own dumps against known good versions of the same software.

It supports stand-alone files and sets in zip files.

History
-------

While looking around for a simple verification tool for rom/iso verification, I found very few that were not built specifically for Windows.

Taking this opportunity to learn something new, I looked at languages other than C to build it in. Go was a natural choice as it has a decent standard library, a good community around it and a healthy amount of decent quality third-party libraries.

Installation
------------

You need a working [Go](https://golang.org/) installation (I used Go 1.10.1 on Ubuntu Linux 18.04)

You will neet to install the required libraries:

    go get github.com/antchfx/xmlquery
    go get github.com/jessevdk/go-flags

You can then build the tool by:

    go install

Usage
-----
    check-roms [OPTIONS] Files...

    Application Options:
    -d, --datfile=               dat file to use as reference database
    -e, --exclude=               extension to exclude from file list
                                 (can be specified multiple times)
    -p, --print=[files|sets|all] which information to print (default: all)
    -r, --rename                 rename unabiguous misnamed files
                                 (only loose files and zipped sets supported)
    -s, --size                   check size on name only match
                                 (helps detect possible under/over-dumps)
    -v, --verbose                show lots more information than is probably necessary

    Help Options:
    -h, --help                   Show this help message

    Arguments:
    Files:                       list of files to check against dat file

Limitations
-----------

- Does not support compression formats other than zip
- Does not rename misnamed files inside zip files 
- Does not detect a set as complete if across multiple zip files
