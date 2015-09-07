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
)

func main() {
	socket := "/tmp/grimreaper.socket"

	_, err := os.Stat(socket)
	if os.IsExist(err) {
		fmt.Printf("Socket still exists: ", err)
		err := os.Remove(socket)
		if err != nil {
			log.Fatal("Cannot remove socket '%v': %v", socket, err)
		}
	}

	l, err := net.Listen("unix", socket)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	defer os.Remove(socket)

	victims := make(map[int]int64)
	go reaper(victims, 5)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("Get client connection error: ", err)
		}

		fmt.Printf("loc %v\n", conn.LocalAddr())
		go handleConnection(conn, victims)
	}
}

func reaper(victims map[int]int64, delay int64) {
	for {
		for key, value := range victims {
			if time.Now().Unix() > value+int64(delay) {
				fmt.Printf("Terminating PID %d", key)
				delete(victims, key)
				syscall.Kill(key, syscall.SIGTERM)
			}
		}
	}
}

func handleConnection(conn net.Conn, victims map[int]int64) {
	for {
		// data, err := bufio.NewReader(conn).ReadString('\n')
		buf := make([]byte, 512)
		nr, err := conn.Read(buf)
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatal("Get client data error: ", err)
		}

		raw := string(buf[0:nr])
		data := strings.Split(raw, ":")
		if len(data) != 2 {
			log.Fatal("Invalid message: ", raw)
		}

		state := data[0]
		pid, err := strconv.ParseInt(data[1], 10, 32)
		if err != nil {
			log.Fatal("Invalid pid: ", data[1])
		}

		if state == "start" {
			victims[int(pid)] = time.Now().Unix()
		} else if state == "done" {
			delete(victims, int(pid))
		} else {
			log.Fatal("Unknown state: ", state)
		}

		//log.Print("%#v\n", data)
		fmt.Printf("%#v\n", raw)
		fmt.Printf("%v\n", victims)

		// conn.Close()
	}
}
