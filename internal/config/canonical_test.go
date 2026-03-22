package config

import (
	"testing"
)

func TestCanonicalFingerprintStable(t *testing.T) {
	a := Default()
	b := Default()
	fp1, err := CanonicalFingerprintSHA256(a)
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := CanonicalFingerprintSHA256(b)
	if err != nil {
		t.Fatal(err)
	}
	if fp1 != fp2 {
		t.Fatalf("identical configs should fingerprint equally: %s vs %s", fp1, fp2)
	}
	b.Control.Mode = "guarded_auto"
	fp3, err := CanonicalFingerprintSHA256(b)
	if err != nil {
		t.Fatal(err)
	}
	if fp3 == fp1 {
		t.Fatal("expected fingerprint to change when config changes")
	}
}
