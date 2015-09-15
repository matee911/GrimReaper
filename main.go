package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Victims represents a collection of PIDs with their timestamps.
type Victims struct {
	sync.RWMutex
	procs map[int]int64
}

// LogLevel describes how verbose logs are.
type LogLevel uint8

// LogLevels definitions.
const (
	CRITICAL LogLevel = 0 << iota
	ERROR    LogLevel = 1
	WARNING  LogLevel = 2
	INFO     LogLevel = 3
	DEBUG    LogLevel = 4
)

var (
	logFile     *os.File
	verbose     string
	logPath     string
	showVersion bool
	socketPath  string
	stdout      bool
	version     string
	victims     = &Victims{procs: make(map[int]int64)}
	logWriters  map[LogLevel]io.Writer

	// Debug is a logger with "DEBUG" level.
	Debug *log.Logger
	// Info is a logger with "INFO" level.
	Info *log.Logger
	// Warning is a logger with "WARNING" level.
	Warning *log.Logger
	// Error is a logger with "ERROR" level.
	Error *log.Logger
	// Critical is a logger with "CRITICAL" level.
	Critical *log.Logger
)

func init() {
	flag.StringVar(&verbose, "verbose", "", "Increase verbosity by passing one or more letters.")
	flag.StringVar(&socketPath, "socket", "/tmp/GrimReaper.socket", "Path to the Unix Domain Socket.")
	flag.StringVar(&logPath, "logpath", "/var/log/GrimReaper.log", "Path to the log file.")
	flag.BoolVar(&stdout, "stdout", false, "Log to stdout/stderr instead of to the log file.")
	flag.BoolVar(&showVersion, "version", false, "print the GrimReaper version information and exit")

	logWriters = map[LogLevel]io.Writer{
		CRITICAL: ioutil.Discard,
		ERROR:    ioutil.Discard,
		WARNING:  ioutil.Discard,
		INFO:     ioutil.Discard,
		DEBUG:    ioutil.Discard,
	}

}

func configureLoggers(writer io.Writer) {
	format := log.Ldate | log.Ltime | log.Lshortfile

	if !stdout {
		writer = logFile
	}

	// Overwrite default writers
	for level := range logWriters {
		if len(verbose) >= int(level) {
			logWriters[level] = writer
		}
	}

	Critical = log.New(logWriters[INFO], "CRITICAL: ", format)
	Error = log.New(logWriters[INFO], "ERROR: ", format)
	Warning = log.New(logWriters[WARNING], "WARNING: ", format)
	Info = log.New(logWriters[INFO], "INFO: ", format)
	Debug = log.New(logWriters[DEBUG], "DEBUG: ", format)
}

func oldSocketExists(socket string) bool {
	if _, err := os.Stat(socket); os.IsNotExist(err) {
		return false
	}
	return true
}

func main() {
	var err error
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	var logWriter io.Writer
	if stdout {
		logWriter = os.Stdout
	} else {
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("Unable to open the log file(%v): %v", logPath, err)
			os.Exit(1)
		}

		logWriter = logFile
		defer logFile.Close()
	}
	configureLoggers(logWriter)

	if oldSocketExists(socketPath) {
		Critical.Fatalf("Socket still exists: %s", socketPath)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}

	defer Info.Println("Shutting down...")
	defer listener.Close()
	defer os.Remove(socketPath)

	go reaper(victims)
	Info.Printf("Ready to accept connections (%s).", socketPath)
	go acceptConnections(listener, victims)

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)
}

func acceptConnections(listener net.Listener, victims *Victims) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			Critical.Fatalf("Got the connection's error: %v", err)
		}
		go handleConnection(conn)
	}
}

func reaper(victims *Victims) {
	for {
		now := time.Now().Unix()

		for pid, deadline := range victims.procs {
			if now > deadline {
				Warning.Printf("Terminating PID %d", pid)
				unregisterProcess(pid)
				go syscall.Kill(pid, syscall.SIGTERM)
			}
		}

		time.Sleep(300 * time.Millisecond)
	}
}

func registerProcess(pid int, timestamp int64, timeout int64) {
	victims.Lock()
	defer victims.Unlock()

	victims.procs[pid] = timestamp + timeout
}

func unregisterProcess(pid int) {
	victims.Lock()
	defer victims.Unlock()

	Debug.Printf("map: %v", victims.procs)
	delete(victims.procs, pid)

}

func handleConnection(conn net.Conn) {
	for {

		buf := make([]byte, 32)
		nr, err := conn.Read(buf)
		if err == io.EOF {
			return
		} else if err != nil {
			Error.Printf("Got data error: %v", err)
		}

		raw := string(buf[0:nr])
		data := strings.Split(raw, ":")
		length := len(data)
		if length < 2 && length > 3 {
			Error.Printf("Invalid message: %v", raw)
		}

		state := data[0]
		pid, err := strconv.ParseInt(data[1], 10, 32)
		if err != nil {
			Error.Printf("Invalid PID: %v", data[1])
		}

		if state == "register" && length == 3 {
			timeout, err := strconv.ParseInt(data[2], 10, 32)
			if err != nil {
				Error.Printf("Invalid timeout: %v", data[2])
			}
			registerProcess(int(pid), time.Now().Unix(), timeout)
		} else if state == "unregister" && length == 2 {
			unregisterProcess(int(pid))
		} else {
			Error.Printf("Invalid message: %v", raw)
		}

		Debug.Printf("Received message: %#v", raw)
	}
}
