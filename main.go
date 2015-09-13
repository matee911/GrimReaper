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

type Victims struct {
	sync.RWMutex
	procs map[int]int64
}

var (
	debug       bool
	logFile     *os.File
	logPath     string
	showVersion bool
	socketPath  string
	stdout      bool
	version     string
	victims     = Victims{procs: make(map[int]int64)}

	Debug    *log.Logger
	Info     *log.Logger
	Warning  *log.Logger
	Error    *log.Logger
	Critical *log.Logger
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Debug mode.")
	flag.StringVar(&socketPath, "socket", "/tmp/GrimReaper.socket", "Path to the Unix Domain Socket.")
	flag.StringVar(&logPath, "logpath", "/var/log/GrimReaper.log", "Path to the log file.")
	flag.BoolVar(&stdout, "stdout", false, "Log to stdout/stderr instead of to the log file.")
	flag.BoolVar(&showVersion, "version", false, "print the GrimReaper version information and exit")
}

func setupLoggers(logFile *os.File) {
	format := log.Ldate | log.Ltime | log.Lshortfile
	debugWriter := ioutil.Discard

	if stdout {
		if debug {
			debugWriter = os.Stdout
		}
		Debug = log.New(debugWriter, "DEBUG: ", format)
		Info = log.New(os.Stdout, "INFO: ", format)
		Warning = log.New(os.Stdout, "WARNING: ", format)
		Error = log.New(os.Stderr, "Error: ", format)
		Critical = log.New(os.Stderr, "CRITICAL: ", format)
	} else {
		if debug {
			debugWriter = logFile
		}
		Debug = log.New(debugWriter, "DEBUG: ", format)
		Info = log.New(logFile, "INFO: ", format)
		Warning = log.New(os.Stdout, "WARNING: ", format)
		Error = log.New(logFile, "ERROR: ", format)
		Critical = log.New(logFile, "CRITICAL: ", format)
	}
}

func removeOldSocket(socket string) {
	_, err := os.Stat(socket)
	if os.IsNotExist(err) {
		return
	} else {
		Critical.Fatalf("Socket still exists: %s", socket)
	}
}

func main() {
	var err error

	flag.Parse()
	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if !stdout {
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("Unable to open the log file(%v): %v", logPath, err)
			os.Exit(1)
		}
		defer logFile.Close()
	}
	setupLoggers(logFile)

	removeOldSocket(socketPath)

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

func acceptConnections(listener net.Listener, victims Victims) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			Critical.Fatalf("Got the connection's error: %v", err)
		}
		go handleConnection(conn)
	}
}

func reaper(victims Victims) {
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
