package main

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestRegisterMessage(t *testing.T) {
	configureLoggers(ioutil.Discard)

	// Non  existing
	processMessage("register:1:10", 1)
	if victims.procs[1] != 11 {
		t.Error("PID with invalid deadline")
	}

	// Non existing - just register
	processMessage("register:2:8", 1)
	if victims.procs[2] != 9 {
		t.Error("PID with invalid deadline")
	}

	// Already existing - overwrite
	processMessage("register:2:5", 2)
	if victims.procs[2] != 7 {
		t.Error("PID with invalid deadline")
	}

	if len(victims.procs) != 2 {
		t.Error("Invalid number of victims")
	}
}

func TestUnregisterMessage(t *testing.T) {
	configureLoggers(ioutil.Discard)

	// Prepare
	processMessage("register:4:10", 1)
	if victims.procs[1] != 11 {
		t.Error("PID with invalid deadline")
	}
	was := len(victims.procs)

	// Unregister now
	processMessage("unregister:4", 1)
	if was-1 != len(victims.procs) {
		t.Error("Invalid number of victims")
	}

	// Try to unregister one more time - nothing should happened
	processMessage("unregister:4", 1)
	if was-1 != len(victims.procs) {
		t.Error("Invalid number of victims")
	}
}

func TestInvalidMessage(t *testing.T) {
	configureLoggers(ioutil.Discard)

	was := len(victims.procs)
	processMessage("register:5:10unregister:6", 1)
	if len(victims.procs) != was {
		t.Error("Changed number of victims")
	}

	processMessage("invalidMessage", 1)
	if len(victims.procs) != was {
		t.Error("Changed number of victims")
	}
}

func BenchmarkUnRegister(b *testing.B) {
	configureLoggers(ioutil.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processMessage(fmt.Sprintf("register:%d:10", i), 1)
		processMessage(fmt.Sprintf("unregister:%d", i), 1)
	}
}
