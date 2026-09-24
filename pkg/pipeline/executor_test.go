package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffrpowell/raggo/pkg/logging"
)

func TestExecuteStageRunsOncePerItem(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "calls.txt")
	e := &Executor{logger: logging.New("test")}

	stage := Stage{
		Name:   "embed",
		Binary: "/bin/sh",
		BuildArgs: func(ctx *ExecutionContext) []string {
			return []string{"-c", `echo "$@" >> "$0"`, out, "-chunk-dir", "c"}
		},
		Items: func(ctx *ExecutionContext) ([]Item, error) {
			return []Item{{ID: "a"}, {ID: "b", Args: []string{"-file-path", "/x/b.pdf"}}}, nil
		},
		ItemFlag: "-document-id",
	}
	if err := e.executeStage(&ExecutionContext{}, stage); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want := "-chunk-dir c -document-id a\n-chunk-dir c -document-id b -file-path /x/b.pdf\n"
	if string(data) != want {
		t.Errorf("got %q, want %q", data, want)
	}
}

func TestExecuteStageStopsOnItemFailure(t *testing.T) {
	e := &Executor{logger: logging.New("test")}
	stage := Stage{
		Name:      "index",
		Binary:    "/bin/sh",
		BuildArgs: func(ctx *ExecutionContext) []string { return []string{"-c", `[ "$2" != bad ]`, "sh"} },
		Items: func(ctx *ExecutionContext) ([]Item, error) {
			return []Item{{ID: "ok"}, {ID: "bad"}, {ID: "never"}}, nil
		},
		ItemFlag: "-episode-id",
	}
	err := e.executeStage(&ExecutionContext{}, stage)
	if err == nil || !strings.Contains(err.Error(), "-episode-id bad") {
		t.Fatalf("expected failure naming the item, got %v", err)
	}
}
