package subprogram

import "testing"

func TestSize(t *testing.T) {
	for i, expected := range map[int]string{
		0: "zero",
	} {
		s := Size(i)
		if expected != s {
			t.Fatalf("Size(%d) = %q , expected %q", i, s, expected)
		}
	}

}
