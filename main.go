package main

/*

TODO:
 - rewrite to use channels
 - parametrized sock path, sigterm-delay, sigkill-delay
 - debug mode

*/

import (
	"io"
	"log"
	"flag"
	"net"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"io/ioutil"
	"syscall"
	"time"
)

type Victims struct {
	sync.RWMutex
	procs map[int]int64
}

var (
	debug bool
	socketPath string
	logPath string
	stdout bool
	logFile *os.File
	victims = Victims{procs: make(map[int]int64)}

	Debug *log.Logger
	Info *log.Logger
	Error *log.Logger
	Critical *log.Logger
)


func init() {
	flag.BoolVar(&debug, "debug", false, "Debug mode.")
	flag.StringVar(&socketPath, "socket", "/tmp/GrimReaper.socket", "Path to the Unix Domain Socket.")
	flag.StringVar(&logPath, "logpath", "/var/log/GrimReaper.log", "Path to the log file.")
	flag.BoolVar(&stdout, "stdout", false, "Log to stdout/stderr instead to the log file.")
}


func setupLoggers(logFile *os.File) {
	format := log.Ldate|log.Ltime|log.Lshortfile
	debugWriter := ioutil.Discard

	fmt.Printf("logFile: %v", logFile)

	if stdout {
		if debug {
			debugWriter = os.Stdout
		}
		Debug = log.New(debugWriter, "DEBUG: ", format)
		Info = log.New(os.Stdout, "INFO: ", format)
		Error = log.New(os.Stderr, "Error: ", format)
		Critical = log.New(os.Stderr, "CRITICAL: ", format)
	} else {
		if debug {
			debugWriter = logFile
		}
		Debug = log.New(debugWriter, "DEBUG: ", format)
		Info = log.New(logFile, "INFO: ", format)
		Error = log.New(logFile, "ERROR: ", format)
		Critical = log.New(logFile, "CRITICAL: ", format)
		Critical.Println("Test")
		logFile.Sync()
	}

	d2 := []byte{115, 111, 109, 101, 10}
	logFile.Write(d2)
	logFile.Sync()
}

func removeOldSocket(socket string) {
	_, err := os.Stat(socket)
	if os.IsNotExist(err) {
		return
	} else {
		Critical.Fatalf("Socket still exists: %s\n", socket)
	}
}


func main() {
	flag.Parse()
	var err error

	if !stdout {
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Cannot open log file(%v): %v", logPath, err)
		}
		defer logFile.Close()
	}
	fmt.Println(logFile)
	setupLoggers(logFile)


	removeOldSocket(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}

	defer log.Println("Shutting down...")
	defer listener.Close()
	defer os.Remove(socketPath)

	go reaper(victims)
	Info.Printf("Ready to accept connections (%s).\n", socketPath)
	logFile.Sync()
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
			Critical.Fatalf("Get client connection error: %v\n", err)
		}

		Debug.Printf("loc %v\n", conn.LocalAddr())
		go handleConnection(conn)
	}
}

func reaper(victims Victims) {
	for {
		now := time.Now().Unix()

		for pid, deadline := range victims.procs {
			if now > deadline {
				Info.Printf("Terminating PID %d\n", pid)
				unregisterProcess(pid)
				go syscall.Kill(pid, syscall.SIGTERM)
			}
		}

		time.Sleep(300 * time.Millisecond)
	}
}


func registerProcess(pid int, timestamp int64, timeout int64){
	victims.Lock()
	victims.procs[pid] = timestamp + timeout
	victims.Unlock()
}

func unregisterProcess(pid int) {
	victims.Lock()
	Debug.Printf("map: %v\n", victims.procs)
	delete(victims.procs, pid)
	victims.Unlock()
}

func handleConnection(conn net.Conn) {
	for {

		// len("open:65000:86400") => 16
		// "close:65000"
		buf := make([]byte, 32)
		nr, err := conn.Read(buf)
		if err == io.EOF {
			return
		} else if err != nil {
			Error.Printf("Get client data error: ", err)
		}

		raw := string(buf[0:nr])
		data := strings.Split(raw, ":")
		length := len(data)
		if length < 2 && length > 3 {
			Error.Printf("Invalid message: ", raw)
		}

		state := data[0]
		pid, err := strconv.ParseInt(data[1], 10, 32)
		if err != nil {
			Error.Printf("Invalid pid: ", data[1])
		}

		if state == "start" && length == 3 {
			timeout, err := strconv.ParseInt(data[2], 10, 32)
			if err != nil {
				Error.Printf("Invalid timeout: ", data[2])
			}
			registerProcess(int(pid), time.Now().Unix(), timeout)
		} else if state == "done" && length == 2 {
			unregisterProcess(int(pid))
		} else {
			Error.Printf("Invalid message: ", raw)
		}

		Debug.Printf("Received message: %#v\n", raw)
	}
}
