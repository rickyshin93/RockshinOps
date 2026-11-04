package fileindex

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func BenchmarkBuild(b *testing.B) {
	root := b.TempDir()
	for i := range 100 {
		dir := filepath.Join(root, fmt.Sprintf("folder-%03d", i))
		if err := os.Mkdir(dir, 0o700); err != nil {
			b.Fatal(err)
		}
		for j := range 50 {
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("report-%03d.txt", j)), nil, 0o600); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		ix := New(root, Rules{})
		if err := ix.Build(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNameCompaction(b *testing.B) {
	source := &Index{children: map[int32][]int32{}, spelled: map[int32][]syllable{}}
	source.add(-1, "", 0, flagDir|flagHidden)
	for i := range 200000 {
		name := fmt.Sprintf("report-%05d.txt", i%80000)
		if i%8 == 0 {
			name = strings.ToUpper(name)
		}
		source.add(0, name, 0, 0)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ix := &Index{entries: slices.Clone(source.entries), names: bytes.Clone(source.names)}
		b.StartTimer()
		ix.compactNames()
	}
}
