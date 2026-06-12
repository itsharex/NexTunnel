package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.env")
	content := "# comment\nCONTROL_PLANE_PORT=9090\nEMPTY=\n RELAY_CONTROL_PORT = 7000 \n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	values, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if values["CONTROL_PLANE_PORT"] != "9090" {
		t.Fatalf("CONTROL_PLANE_PORT = %q", values["CONTROL_PLANE_PORT"])
	}
	if values["RELAY_CONTROL_PORT"] != "7000" {
		t.Fatalf("RELAY_CONTROL_PORT = %q", values["RELAY_CONTROL_PORT"])
	}
	if _, ok := values["# comment"]; ok {
		t.Fatal("comment line should be ignored")
	}
}
