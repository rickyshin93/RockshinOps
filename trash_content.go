package main

import (
	"context"
	"crypto/sha256"
	"io"
	"io/fs"
	"os"
)

type trashContentReader struct {
	buffer [32 * 1024]byte
}

type trashContextReader struct {
	ctx context.Context
	io.Reader
}

func (reader trashContextReader) Read(bytes []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.Reader.Read(bytes)
}

func sameFileState(before, after fs.FileInfo) bool {
	return sameFileMetadata(before, after) && fileChangeTime(before) == fileChangeTime(after)
}

func (reader *trashContentReader) hash(ctx context.Context, root *os.Root, path string, expected fs.FileInfo) ([sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	file, err := root.OpenFile(path, trashReadFlags, 0)
	if err != nil {
		return digest, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return digest, err
	}
	if !info.Mode().IsRegular() || !sameFileState(expected, info) {
		return digest, errTrashReviewChanged
	}
	hash := sha256.New()
	if _, err := io.CopyBuffer(hash, trashContextReader{ctx: ctx, Reader: file}, reader.buffer[:]); err != nil {
		return digest, err
	}
	after, err := file.Stat()
	if err != nil {
		return digest, err
	}
	if !sameFileState(info, after) {
		return digest, errTrashReviewChanged
	}
	hash.Sum(digest[:0])
	return digest, nil
}
