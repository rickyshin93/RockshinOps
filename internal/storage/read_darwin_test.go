package storage

import (
	"testing"
)

func TestNativeCapacity(t *testing.T) {
	v, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	if v.Total == 0 || v.Available > v.Total || v.Mount == "" || v.Format == "" {
		t.Fatalf("invalid native sample: %+v", v)
	}
	t.Logf("native sample: mount=%s format=%s total=%d available=%d", v.Mount, v.Format, v.Total, v.Available)
}
