// Killer - Repeatedly try to kill a process
//
// Copyright 2017 Nick Pillitteri
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"syscall"
	"time"
)

const (
	DEFAULT_INTERVAL = 1
	DEFAULT_TIMEOUT  = 30
)

func processExists(p *os.Process) bool {
	res := syscall.Kill(p.Pid, syscall.Signal(0))
	return res == nil || res != syscall.ESRCH
}

func killNicely(p *os.Process, interval int, timeout int) (bool, error) {
	elapsed := 0

	for {
		// Note that we use `syscall.Kill` instead of `p.Signal` so that we don't
		// get the "process already finished" error which I can't figure out how to
		// actually test for. This in turn results in spending the entire `timeout`
		// waiting for something to stop that has already stopped.
		res := syscall.Kill(p.Pid, syscall.SIGTERM)
		if res == syscall.ESRCH {
			return true, nil
		} else if res == syscall.EPERM {
			return false, res
		} else if res == syscall.EINVAL {
			return false, res
		}

		if elapsed >= timeout {
			return false, nil
		}

		time.Sleep(time.Duration(interval * 1e9))
		elapsed += interval
	}
}

func killNotSoNicely(p *os.Process) error {
	res := syscall.Kill(p.Pid, syscall.SIGKILL)
	// Successfully sent the signal or it's already stopped
	if res == nil || res == syscall.ESRCH {
		return nil
	}

	return res
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [PID]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Repeatedly try to stop a process with SIGTERM and eventually SIGKILL\n\n")
		flag.PrintDefaults()
	}

	interval := flag.Int("interval", DEFAULT_INTERVAL, "How long to wait between attempts to stop a process in seconds")
	timeout := flag.Int("timeout", DEFAULT_TIMEOUT, "How long to wait total when trying to stop a process in seconds")
	disableKill := flag.Bool("disable-kill", false, "Disable use of SIGKILL as a last resort when stopping a process")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("PID is required")
	}

	pid, err := strconv.ParseInt(flag.Args()[0], 10, 32)
	if err != nil {
		log.Fatal(err)
	}

	p, err := os.FindProcess(int(pid))
	if err != nil {
		log.Fatal(err)
	}

	// Exit with an error if this process doesn't even exist before we attempt to stop it
	if !processExists(p) {
		log.Fatalf("Process %d does not exist\n", pid)
	}

	stopped, err := killNicely(p, *interval, *timeout)
	if err != nil {
		log.Fatal(err)
	}

	if !stopped && !*disableKill {
		if err := killNotSoNicely(p); err != nil {
			log.Fatal(err)
		}
	} else if !stopped {
		log.Fatalf("Failed to stop %d before timeout\n", pid)
	}
}
