package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestCommandHelpAndErrors(t *testing.T) {
	for _, test := range []struct {
		args    []string
		want    string
		failure bool
	}{
		{[]string{"--help"}, "Spec review is the default", false},
		{[]string{"audit", "--help"}, "checkpoint", false},
		{[]string{"audit", "build", "--help"}, "--strict", false},
		{[]string{"audit", "unknown"}, "Unknown audit command", true},
		{[]string{"audit", "init"}, "--project is required", true},
		{[]string{"-port", "-1"}, "port must", true},
		{[]string{"unexpected"}, "unexpected argument", true},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			var out, diagnostics bytes.Buffer
			err := run(t.Context(), test.args, &out, &diagnostics)
			if (err != nil) != test.failure {
				t.Fatalf("error = %v", err)
			}
			text := out.String() + diagnostics.String()
			if err != nil {
				text += err.Error()
			}
			if !strings.Contains(text, test.want) {
				t.Fatalf("output lacks %q: %s", test.want, text)
			}
		})
	}
}

func TestBinaryModesWithoutNode(t *testing.T) {
	binary := os.Getenv("SPECTRE_BINARY")
	if binary == "" {
		t.Skip("run through task test or task test:audit to exercise the built executable")
	}
	dir := t.TempDir()
	project := filepath.Join(dir, "project")
	specDir := filepath.Join(dir, "spec")
	emptyPath := filepath.Join(dir, "empty-path")
	for _, path := range []string{project, specDir, emptyPath} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	env := append(os.Environ(), "PATH="+emptyPath, "AUDIT_REPORT_HOME="+filepath.Join(dir, "index"), "XDG_CACHE_HOME="+filepath.Join(dir, "cache"))
	command := func(args ...string) *exec.Cmd {
		cmd := exec.Command(binary, args...)
		cmd.Dir = dir
		cmd.Env = env
		return cmd
	}
	runCommand := func(args ...string) []byte {
		t.Helper()
		out, err := command(args...).CombinedOutput()
		if err != nil {
			t.Fatalf("spectre %v: %v\n%s", args, err, out)
		}
		return out
	}
	starter := filepath.Join(dir, "starter.json")
	runCommand("audit", "build", "--init", starter)
	runCommand("audit", "build", starter, "--strict", "-o", filepath.Join(dir, "report.html"))
	var created struct {
		AuditID string `json:"auditId"`
		Paths   struct {
			Report string `json:"report"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(runCommand("audit", "init", "--project", project, "--source", starter), &created); err != nil {
		t.Fatal(err)
	}
	if created.AuditID == "" {
		t.Fatal("audit did not initialize")
	}
	runCommand("audit", "resume", "--project", project)
	entries, err := os.ReadDir("../../internal/spec/testdata/atc-spec")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join("../../internal/spec/testdata/atc-spec", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(specDir, entry.Name()), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		name  string
		args  []string
		audit bool
		want  string
	}{
		{"default spec", []string{"-dir", specDir, "-port", "0"}, false, "ATC Spec Review"},
		{"explicit spec", []string{"spec", "-dir", specDir, "-port", "0"}, false, "ATC Spec Review"},
		{"audit", []string{"audit", "view", "--project", project}, true, "Password reset links never expire"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, test.args...)
			cmd.Dir = dir
			cmd.Env = env
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			waited := false
			t.Cleanup(func() {
				if !waited {
					cmd.Process.Kill()
					cmd.Wait()
				}
			})
			var address string
			if test.audit {
				var result struct {
					URL string `json:"url"`
				}
				if err := json.NewDecoder(stdout).Decode(&result); err != nil {
					t.Fatal(err)
				}
				address = result.URL
			} else {
				scanner := bufio.NewScanner(stderr)
				pattern := regexp.MustCompile(`url=(http://\S+)`)
				for scanner.Scan() {
					if m := pattern.FindStringSubmatch(scanner.Text()); len(m) == 2 {
						address = m[1]
						break
					}
				}
				if address == "" {
					t.Fatal("spec server did not print URL", scanner.Err())
				}
			}
			response, err := http.Get(address)
			if err != nil {
				t.Fatal(err)
			}
			page, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil || response.StatusCode != 200 || !bytes.Contains(page, []byte(test.want)) {
				t.Fatalf("page status %d, error %v, missing %q", response.StatusCode, err, test.want)
			}
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Wait(); err != nil {
				t.Fatal(err)
			}
			waited = true
		})
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(created.Paths.Report), ".viewer.lock")); !os.IsNotExist(err) {
		t.Fatal("audit viewer lock survived interrupt")
	}
}
