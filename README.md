# Grim Reaper - the process killer

![Build Status](https://travis-ci.org/matee911/GrimReaper.svg)

------

GrimReaper is a process killer, originally written to solve a problem of too long processed requests by the Django/flup.
Any process can communicate with GrimReaper by the unix domain socket, and send the command to register itself (or other process) with the time (and the PID).
If the GrimReaper doesn't receive the unregister command (with the same PID) before time passes, it will kill that process.


## Who uses Grim Reaper

* [ipla.tv](http://ipla.tv/)

## Usage

   Usage of GrimReaper:
     -debug
       	Debug mode.
     -logpath string
       	Path to the log file. (default "/var/log/GrimReaper.log")
     -socket string
       	Path to the Unix Domain Socket. (default "/tmp/GrimReaper.socket")
     -stdout
       	Log to stdout/stderr instead of to the log file.

## Clients

### Python

* [GrimReapersPie](http://github.com/matee911/GrimReapersPie)

## Credits

* [Kamil Essekkat](https://github.com/ekamil) Project's name
