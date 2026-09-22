package s3

import (
	"path/filepath"
	"testing"
)

func TestNormalizeObjectKey(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{
			name: "simple relative",
			in:   "expense_export/task.xlsx",
			want: "expense_export/task.xlsx",
		},
		{
			name: "trim spaces",
			in:   "  a/b.txt  ",
			want: "a/b.txt",
		},
		{
			name: "clean redundant dot segments",
			in:   "a/./b/c.txt",
			want: "a/b/c.txt",
		},
		{
			name: "backslash to slash",
			in:   `a\b\c.txt`,
			want: "a/b/c.txt",
		},
		{
			name:    "empty",
			in:      "",
			wantErr: ErrInvalidObjectKey,
		},
		{
			name:    "whitespace only",
			in:      "   ",
			wantErr: ErrInvalidObjectKey,
		},
		{
			name:    "dot only",
			in:      ".",
			wantErr: ErrInvalidObjectKey,
		},
		{
			name:    "parent traversal prefix",
			in:      "../secret.txt",
			wantErr: ErrInvalidObjectKey,
		},
		{
			name: "parent collapsed by clean",
			in:   "a/../b.txt",
			want: "b.txt",
		},
		{
			name:    "still escapes after clean",
			in:      "foo/../../secret.txt",
			wantErr: ErrInvalidObjectKey,
		},
		{
			name:    "leading slash",
			in:      "/etc/passwd",
			wantErr: ErrInvalidObjectKey,
		},
	}

	abs := filepath.Join(t.TempDir(), "outside.txt")
	tests = append(tests, struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		name:    "os absolute path",
		in:      abs,
		wantErr: ErrInvalidObjectKey,
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeObjectKey(tc.in)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("err=%v want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
