package runCommand

import (
	"testing"
)

func TestMarshal(t *testing.T) {

	tc := NewExecData([]string{"ls", "-lta"}, "..", "")
	err := tc.Run(func(out string, err string) {
		t.Fatalf(out)
	})
	if err != nil {
		t.Fatalf("Error:%e", err)
	}
}
