package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectIDsMergesFlagAndFileWithoutDuplicates(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "ids.txt")
	os.WriteFile(file, []byte("# front page\nb\n\nc\na\n"), 0o644)

	ids, err := collectIDs("a, b", file)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c"}
	if len(ids) != len(want) {
		t.Fatalf("ids %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ids %v, want %v", ids, want)
		}
	}
}

func TestCollectIDsIsEmptyWithNothingGiven(t *testing.T) {
	ids, err := collectIDs("", "")
	if err != nil || len(ids) != 0 {
		t.Fatalf("ids %v err %v", ids, err)
	}
}
