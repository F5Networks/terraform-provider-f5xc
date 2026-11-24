//go:build unix

//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package main

import (
	"bufio"
	"flag"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestMain_DebugServeStopsOnInterrupt verifies that main with -debug starts
// the provider server in managed debug mode, prints the reattach
// configuration, and returns without error after an interrupt signal.
func TestMain_DebugServeStopsOnInterrupt(t *testing.T) {
	origArgs, origFlags, origStdout := os.Args, flag.CommandLine, os.Stdout
	t.Cleanup(func() {
		os.Args, flag.CommandLine, os.Stdout = origArgs, origFlags, origStdout
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })

	os.Args = []string{"terraform-provider-f5xc", "-debug"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		defer close(done)
		main()
	}()

	// Send the interrupt only after the server prints its reattach
	// configuration, because the signal handler is registered by then.
	started := make(chan struct{})
	go func() {
		sc := bufio.NewScanner(r)
		signalled := false
		for sc.Scan() {
			if !signalled && strings.Contains(sc.Text(), "Provider started") {
				signalled = true
				close(started)
			}
		}
	}()

	select {
	case <-started:
	case <-time.After(30 * time.Second):
		t.Fatal("provider did not start in debug mode")
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("main did not return after interrupt")
	}
	_ = w.Close()
}
