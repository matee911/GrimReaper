package main

import (
	"errors"
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

// Stats collects informations and statistics about the GrimReaper.
type Stats struct {
	sync.RWMutex
	startTime           int64
	registerCalls       int64
	unregisterCalls     int64
	pingCalls           int64
	statsCalls          int64
	kills               int64
	invalidCommandCalls int64
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
	stats       = &Stats{}
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

	stats.Lock()
	stats.startTime = time.Now().Unix()
	stats.Unlock()

	go reaper(victims)
	go statsLogger(stats)
	Info.Printf("Ready to accept connections (%s).", socketPath)
	go acceptConnections(listener, victims)

	// Finish execution after SIGINT or SIGTERM is received.
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
func statsLogger(stats *Stats) {
	for {
		Info.Printf("Uptime %ds/%dd; Kills: %d; Register: %d Unregister: %d; Ping: %d; Stats: %d; Invalid: %d",
			time.Now().Unix()-stats.startTime,
			(time.Now().Unix()-stats.startTime)/86400,
			stats.kills,
			stats.registerCalls,
			stats.unregisterCalls,
			stats.pingCalls,
			stats.statsCalls,
			stats.invalidCommandCalls,
		)
		time.Sleep(60 * time.Second)
	}
}

func reaper(victims *Victims) {
	for {
		now := time.Now().Unix()

		for pid, deadline := range victims.procs {
			if now > deadline {
				Warning.Printf("Terminating PID %d", pid)
				unregisterProcess(pid)
				stats.Lock()
				stats.kills++
				stats.Unlock()
				go syscall.Kill(pid, syscall.SIGTERM)
			}
		}

		time.Sleep(300 * time.Millisecond)
	}
}

func registerProcess(pid int, timestamp int64, timeout int64) bool {
	victims.Lock()
	defer victims.Unlock()

	victims.procs[pid] = timestamp + timeout
	return true
}

func unregisterProcess(pid int) bool {
	victims.Lock()
	defer victims.Unlock()

	Debug.Printf("map: %v", victims.procs)
	delete(victims.procs, pid)
	return true
}

func registerCommandCall(args []string, currentTimestamp int64) (err error) {
	if len(args) != 2 {
		return errors.New("Bad number of arguments")
	}

	pid, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return err
	}

	timeout, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return err
	}

	if !registerProcess(int(pid), currentTimestamp, timeout) {
		return errors.New("Cannot register process")
	}
	return
}

func unregisterCommandCall(args []string) (err error) {
	if len(args) != 1 {
		return errors.New("Bad number of arguments")
	}

	pid, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return err
	}

	if !unregisterProcess(int(pid)) {
		return errors.New("Cannot unregister process")
	}
	return
}

func processMessage(raw string, currentTimestamp int64) (message string, err error) {
	Info.Printf("Received message: %#v", raw)
	/*
		Accepted commands:

		ping
		register:<PID>:<timeout>
		stats
		unregister:<PID>

		Commands enhancements proposals:

		register:<PID>:<timeout>:<optional additional message - url
	*/
	data := strings.Split(raw, ":")
	command := data[0]

	stats.Lock()
	defer stats.Unlock()

	switch command {
	case "register":
		stats.registerCalls++
		err = registerCommandCall(data[1:], currentTimestamp)
		if err == nil {
			message = "OK"
		} else {
			message = "ERROR"
		}
	case "unregister":
		stats.unregisterCalls++
		err = unregisterCommandCall(data[1:])
		if err == nil {
			message = "OK"
		} else {
			message = "ERROR"
		}
	case "ping":
		stats.pingCalls++
		message = "OK"
	case "stats":
		stats.statsCalls++
		message = "OK"
	default:
		stats.invalidCommandCalls++
		message = "ERROR"
		err = errors.New("Unknown command")
	}
	return
}

func handleConnection(conn net.Conn) {
	for {

		// TODO(matee): Split messages by EOL
		buf := make([]byte, 32)
		nr, err := conn.Read(buf)
		if err == io.EOF {
			return
		} else if err != nil {
			Error.Printf("Got data error: %v", err)
		}

		raw := string(buf[0:nr])
		message, err := processMessage(raw, time.Now().Unix())
		Info.Printf("%s: %s", message, err)
		// TODO(matee): Send message as response
	}
}
