package main

/*

TODO:
 - rewrite to use channels
 - parametrized sock path, sigterm-delay, sigkill-delay
 - debug mode

*/

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"sync"
)

type Victims struct {
	sync.RWMutex
	procs map[int]int64
}

var victims = Victims{procs: make(map[int]int64)}


func removeOldSocket(socket string) {
	// remove old socket
	// todo: check removing with defer
	_, err := os.Stat(socket)
	if os.IsExist(err) {
		fmt.Printf("Socket still exists: ", err)
		err := os.Remove(socket)
		if err != nil {
			log.Fatal("Cannot remove socket '%v': %v", socket, err)
		}
	}
}

func main() {
	socket := "/tmp/grimreaper.socket"

	removeOldSocket(socket)

	listener, err := net.Listen("unix", socket)
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	defer os.Remove(socket)

	go reaper(victims)
	acceptConnections(listener, victims)
}


func acceptConnections(listener net.Listener, victims Victims) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("Get client connection error: ", err)
		}

		fmt.Printf("loc %v\n", conn.LocalAddr())
		go handleConnection(conn)
	}
}

func reaper(victims Victims) {
	for {

		now := time.Now().Unix()

		for pid, deadline := range victims.procs {
			if now > deadline {
				fmt.Printf("Terminating PID %d", pid)
				unregisterProcess(pid)
				go syscall.Kill(pid, syscall.SIGTERM)
			}
		}
	}
}


func registerProcess(pid int, timestamp int64, timeout int64){
	victims.Lock()
	victims.procs[pid] = timestamp + timeout
	victims.Unlock()
}

func unregisterProcess(pid int) {
	victims.Lock()
	fmt.Printf("map: %v", victims.procs)
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
			log.Fatal("Get client data error: ", err)
		}

		raw := string(buf[0:nr])
		data := strings.Split(raw, ":")
		length := len(data)
		if length < 2 && length > 3 {
			log.Fatal("Invalid message: ", raw)
		}

		state := data[0]
		pid, err := strconv.ParseInt(data[1], 10, 32)
		if err != nil {
			log.Fatal("Invalid pid: ", data[1])
		}

		if state == "start" && length == 3 {
			timeout, err := strconv.ParseInt(data[2], 10, 32)
			if err != nil {
				log.Fatal("Invalid pid: ", data[2])
			}
			registerProcess(int(pid), time.Now().Unix(), timeout)
		} else if state == "done" && length == 2 {
			unregisterProcess(int(pid))
		} else {
			log.Fatal("Invalid message: ", raw)
		}

		//log.Print("%#v\n", data)
		fmt.Printf("%#v\n", raw)
	}
}
