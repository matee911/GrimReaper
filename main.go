package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/textproto"
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
	registerCalls       uint64
	unregisterCalls     uint64
	pingCalls           uint64
	statsCalls          uint64
	kills               uint64
	invalidCommandCalls uint64
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
	forceCleanup bool
	logFile      *os.File
	logPath      string
	showVersion  bool
	socketPath   string
	stdout       bool
	verbose      string
	version      string

	stats   = &Stats{}
	victims = &Victims{procs: make(map[int]int64)}

	logWriters map[LogLevel]io.Writer

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
	flag.BoolVar(&showVersion, "version", false, "print the GrimReaper version information and exit.")
	flag.BoolVar(&forceCleanup, "force-cleanup", false, "Force removing socket file before start.")

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
		if forceCleanup {
			if os.Remove(socketPath); err != nil {
				Critical.Fatalf("Cannot remove existing socket: %s", socketPath)
			}
		} else {
			Critical.Fatalf("Socket still exists: %s", socketPath)
		}
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
	Info.Printf("Starting GrimReaper %s", version)

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

func killPid(pid int) {
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		Error.Printf("Killing the process: %v", err)
	}
}

func reaper(victims *Victims) {
	for {
		now := time.Now().Unix()

		for pid, deadline := range victims.procs {
			if now > deadline {
				Warning.Printf("Terminating PID: %d", pid)
				unregisterProcess(pid)
				stats.Lock()
				stats.kills++
				stats.Unlock()
				go killPid(pid)
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
		return fmt.Errorf("Bad number of arguments: %v", args)
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
		return fmt.Errorf("Cannot register the process: %v", pid)
	}
	return
}

func unregisterCommandCall(args []string) (err error) {
	if len(args) != 1 {
		return fmt.Errorf("Bad number of arguments: %v", args)
	}

	pid, err := strconv.ParseInt(args[0], 10, 32)
	if err != nil {
		return err
	}

	if !unregisterProcess(int(pid)) {
		return fmt.Errorf("Cannot unregister the process: %v", pid)
	}
	return
}

func processMessage(rawMessage string, currentTimestamp int64) (message string, err error) {
	Debug.Printf("Received message: %q", rawMessage)
	/*
		Accepted commands:

		ping
		register:<PID>:<timeout>
		stats
		unregister:<PID>

		Commands enhancements proposals:

		register:<PID>:<timeout>:<optional additional message - url
	*/
	data := strings.Split(rawMessage, ":")
	command := data[0]

	stats.Lock()
	defer stats.Unlock()

	switch command {
	case "register":
		err = registerCommandCall(data[1:], currentTimestamp)
		if err == nil {
			message = "OK"
			stats.registerCalls++
		} else {
			message = "ERROR"
			stats.invalidCommandCalls++
		}
	case "unregister":
		err = unregisterCommandCall(data[1:])
		if err == nil {
			message = "OK"
			stats.unregisterCalls++
		} else {
			message = "ERROR"
			stats.invalidCommandCalls++
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
		err = fmt.Errorf("Unknown command: %q", rawMessage)
	}
	return
}

func handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)

	for {
		if line, err := tp.ReadLine(); err != nil {
			Error.Printf("Got data error: %v", err)
		} else {
			Error.Printf("line: %s", line)
			message, err := processMessage(line, time.Now().Unix())
			if err != nil {
				Error.Printf("%s: %s", message, err)
			}
		}
	}
}
