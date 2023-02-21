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

You need a working [Go](https://golang.org/) installation (I use Go 1.20 on Ubuntu Linux 20.04 LTS)

Build the tool with:

    go install

Usage
-----
    Usage:
      check-roms [OPTIONS] <audit | check | lookup | zip>
    
    Application Options:
      -d, --datfile=                      dat file to use as reference database
      -l, --level=[error|warn|info|debug] level for information to show (default: error)
    
    Help Options:
      -h, --help                          Show this help message
    
    Available commands:
      audit                               Audit files against datfile
      check                               Check files against datfile
      lookup                              Lookup a datfile rom entry
      zip                                 Zip complete roms into sets

    [audit command options]
      -e, --exclude=                      extension to exclude from file list (can
                                          be specified multiple times)
      -m, --method=[sha1|md5|crc]         method to use to match roms (default:
                                          sha1)
      -r, --rename                        rename unambiguous misnamed files (only
                                          loose files and zipped sets supported)
      -w, --workers=                      number of concurrent workers to use
                                          (default: 10)

    [audit command arguments]
      OutputFile:                         audit file for output (default:
                                          audit_<timestamp>.txt)

    [check command options]
      -a, --allsets                       report all sets that are missing
      -e, --exclude=                      extension to exclude from file list (can be
                                          specified multiple times)
      -m, --method=[sha1|md5|crc]         method to use to match roms (default: sha1)
      -r, --rename                        rename unambiguous misnamed files (only loose
                                          files and zipped sets supported)
      -w, --workers=                      number of concurrent workers to use (default:

    [check command arguments]
      Files:                              list of files to check against dat file (default: *)

    [lookup command options]
      -k, --key=[name|crc|md5|sha1]       key to use for lookup (ignored for game mode) (default: name)
      -m, --mode=[rom|game]               element to lookup (default: rom)
      -x, --exact                         use exact match (otherwise use substring match)
      
    [lookup command arguments]
      Keys:                               list of keys to lookup

    [zip command options]
      -e, --exclude=                      extension to exclude from file list (can be specified multiple times)
      -i, --infozip                       use info-zip command line tool instead of internal zip function
      -o, --outdir=                       directory in which to output zipped files (default: .)
      -m, --remove                        remove files after zipping

    [zip command arguments]
      Files:                              list of files to check and zip (default: *)

Limitations
-----------

- Does not support compression formats other than zip.
- Does not rename misnamed files inside zip files.
- Does not read elements other than `<rom>` inside `<game>` as I  am yet to find a dat file containing these.
- 7-zip complains that large zipped files have errors when internal go zip functionality is used. No other tool has this problem.
