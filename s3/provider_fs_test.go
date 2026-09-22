package s3

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderFs_SaveOpenRemove(t *testing.T) {
	root := filepath.Join(t.TempDir(), "uploads")
	p, err := NewProviderFs(Region(root))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	key := "expense_export/task-1.xlsx"
	body := []byte("xlsx-content")
	n, err := p.Save(ctx, key, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(body)) {
		t.Fatalf("written=%d want %d", n, len(body))
	}
	var got []byte
	err = p.Open(ctx, key, func(r io.Reader) error {
		var e error
		got, e = io.ReadAll(r)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("read mismatch")
	}
	if err = p.Remove(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, key)); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, err=%v", err)
	}
}

func TestProviderFs_RejectTraversal(t *testing.T) {
	p, err := NewProviderFs(Region(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Save(context.Background(), "../escape.txt", bytes.NewReader(nil))
	if err != ErrInvalidObjectKey {
		t.Fatalf("want ErrInvalidObjectKey, got %v", err)
	}
}
