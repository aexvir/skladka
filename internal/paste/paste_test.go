package paste_test

import (
	"testing"

	"github.com/aexvir/skladka/internal/paste"
	"github.com/stretchr/testify/assert"
)

func TestPasteFileName(t *testing.T) {
	tests := map[string]struct {
		paste paste.Paste
		want  string
	}{
		"untitled with no mimetype": {
			paste: paste.Paste{
				Title: "untitled",
			},
			want: "untitled.txt",
		},
		"untitled with unknown mimetype": {
			paste: paste.Paste{
				Title:    "untitled",
				Mimetype: ptr(paste.MimeUknown),
			},
			want: "untitled.txt",
		},
		"untitled with known mimetype": {
			paste: paste.Paste{
				Title:    "untitled",
				Mimetype: ptr(paste.MimeWebp),
			},
			want: "untitled.webp",
		},
		"title with extension no mimetype": {
			paste: paste.Paste{
				Title: "title.abc",
			},
			want: "title.abc",
		},
		"title with extension with mimetype": {
			paste: paste.Paste{
				Title:    "title.abc",
				Mimetype: ptr(paste.MimeWebp),
			},
			want: "title.abc",
		},
	}

	for name, test := range tests {
		t.Run(
			name,
			func(t *testing.T) {
				got := test.paste.FileName()
				assert.Equal(t, test.want, got)
			},
		)
	}
}

func ptr[t any](v t) *t {
	return &v
}
