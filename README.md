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

You will need to install the required libraries:

    go get github.com/antchfx/xmlquery
    go get github.com/jessevdk/go-flags

You can then build the tool by:

    go install

Usage
-----
    check-roms [OPTIONS] <check | lookup | zip>

    Application Options:
    -d, --datfile=                dat file to use as reference database
                                  (can be specified in a file named ".dat" in the
                                  current directory instead for ease of use)
    -v, --verbose                 show lots more information than is probably necessary

    Help Options:
    -h, --help                    Show this help message

    Available commands:
    check                         Check files against datfile
    lookup                        Lookup a datfile rom entry
    zip                           Zip complete roms into sets

    [check command options]
    -e, --exclude=                extension to exclude from file list (can be specified multiple times)
    -p, --print=[files|sets|all]  which information to print (default: all)
    -r, --rename                  rename unambiguous misnamed files (only loose files and zipped sets supported)

    [check command arguments]
    Files:                        list of files to check against dat file

    [lookup command options]
    -k, --key=[crc|md5|name|sha1] key to use for lookup (ignored for game mode) (default: name)
    -m, --mode=[rom|game]         element to lookup (default: rom)
    -x, --exact                   use exact match (otherwise use substring match)

    [lookup command arguments]
    Keys:                         list of keys to lookup

    [zip command options]
    -e, --exclude=                extension to exclude from file list (can be specified multiple times)
    -i, --infozip  use info-zip command line tool instead of internal zip function

    [zip command arguments]
    Files:                        list of files to check and zip


Limitations
-----------

- Does not support compression formats other than zip
- Does not rename misnamed files inside zip files 
- Does not detect a set as complete if across multiple zip files
- Does not read elements other than `<rom>` inside `<game>` as I  am yet to find a dat file containing these 
- 7-zip complains that large zipped files have errors when internal go zip functionality is used. No other tool has this problem.
