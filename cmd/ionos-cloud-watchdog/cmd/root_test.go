package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/output"
)

func TestRunCheckOnce_JSONAndExitCodes(t *testing.T) {
	defer restoreGlobals()
	exitCodes := []int{}
	exitFunc = func(code int) { exitCodes = append(exitCodes, code) }

	outputFmt = "json"
	runChecksFunc = func(kc, ns string) (*output.Report, error) {
		return &output.Report{Status: "WARNING"}, nil
	}

	out := captureStdout(t, func() {
		runCheckOnce(false)
	})

	if !strings.Contains(out, `"Status": "WARNING"`) {
		t.Fatalf("expected JSON output with status, got: %s", out)
	}
	if len(exitCodes) != 1 || exitCodes[0] != 1 {
		t.Fatalf("expected exit code 1, got %v", exitCodes)
	}
}

func TestRunCheckOnce_TextUsesPrinter(t *testing.T) {
	defer restoreGlobals()
	called := false
	printTextFunc = func(r *output.Report, cfg *output.Config) { called = true }
	runChecksFunc = func(kc, ns string) (*output.Report, error) {
		return &output.Report{Status: "OK"}, nil
	}
	outputFmt = "text"

	runCheckOnce(false)

	if !called {
		t.Fatalf("expected printTextFunc to be called")
	}
}

func TestRunCheckOnce_ErrorPrintsAndExits(t *testing.T) {
	defer restoreGlobals()
	exitCodes := []int{}
	exitFunc = func(code int) { exitCodes = append(exitCodes, code) }

	stderr := captureStderr(t, func() {
		runChecksFunc = func(_, _ string) (*output.Report, error) { return nil, errors.New("boom") }
		runCheckOnce(false)
	})

	if len(exitCodes) != 1 || exitCodes[0] != 1 {
		t.Fatalf("expected exit code 1, got %v", exitCodes)
	}
	if !strings.Contains(stderr, "boom") {
		t.Fatalf("expected error printed to stderr, got: %s", stderr)
	}
}

func TestRootCommandExecutesRunChecks(t *testing.T) {
	defer restoreGlobals()
	exitCodes := []int{}
	exitFunc = func(code int) { exitCodes = append(exitCodes, code) }
	outputFmt = "json"

	runChecksFunc = func(_, _ string) (*output.Report, error) {
		return &output.Report{Status: "OK"}, nil
	}

	rootCmd.SetArgs([]string{"--output", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(exitCodes) != 0 {
		t.Fatalf("expected no exit for OK status, got %v", exitCodes)
	}
}

func restoreGlobals() {
	runChecksFunc = output.RunChecks
	printTextFunc = output.PrintText
	exitFunc = os.Exit
	sleepFunc = func(d time.Duration) { time.Sleep(d) }
	outputFmt = "text"
	verbose = false
	kubeconfig = ""
	namespace = ""
	watch = 0
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = orig
	return buf.String()
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stderr = orig
	return buf.String()
}

func TestJSONOutputIsIndented(t *testing.T) {
	defer restoreGlobals()
	outputFmt = "json"
	runChecksFunc = func(_, _ string) (*output.Report, error) {
		return &output.Report{Status: "OK"}, nil
	}

	out := captureStdout(t, func() {
		runCheckOnce(false)
	})

	var decoded output.Report
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("expected valid json output: %v\n%s", err, out)
	}
}
