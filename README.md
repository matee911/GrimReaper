# Grim Reaper - the process killer

[![Build Status](https://travis-ci.org/matee911/GrimReaper.svg)](https://travis-ci.org/matee911/GrimReaper)
[![GitHub issues](https://img.shields.io/github/issues/matee911/GrimReaper.svg)](https://github.com/matee911/GrimReaper/issues)
[![GitHub license](https://img.shields.io/github/license/matee911/GrimReaper.svg)](https://github.com/matee911/GrimReaper/blob/master/LICENSE)
[![GitHub tag](https://img.shields.io/github/tag/matee911/GrimReaper.svg)]()
[![GitHub release](https://img.shields.io/github/release/matee911/GrimReaper.svg)]()
[![GitHub commits](https://img.shields.io/github/commits-since/matee911/GrimReaper/0.1.0a1.svg)]()
[![Twitter](https://img.shields.io/twitter/url/https/github.com/matee911/GrimReaper.svg?style=social)](https://twitter.com/intent/tweet?text=Wow:&url=%5Bobject%20Object%5D)


------

GrimReaper is a process killer, originally written to solve a problem of too long processed requests by the Django/flup.
Any process can communicate with GrimReaper by the unix domain socket, and send the command to register itself (or other process) with the time (and the PID).
If the GrimReaper doesn't receive the unregister command (with the same PID) before time passes, it will kill that process.


## Who uses Grim Reaper

* [ipla.tv](http://ipla.tv/)

## Usage

```bash
Usage of GrimReaper:
  -debug
    	Debug mode.
  -logpath string
    	Path to the log file. (default "/var/log/GrimReaper.log")
  -socket string
    	Path to the Unix Domain Socket. (default "/tmp/GrimReaper.socket")
  -stdout
    	Log to stdout/stderr instead of to the log file.
  -version
    	print the GrimReaper version information and exit
```

## Clients

### Python

* [GrimReapersPie](http://github.com/matee911/GrimReapersPie)

## Credits

* [Kamil Essekkat](https://github.com/ekamil) Project's name
* [Kamil Dzięcioł](https://github.com/woodpeaker) Code reviews
