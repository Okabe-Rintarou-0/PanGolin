package parser

import (
	"reflect"
	"testing"
)

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:  "simple",
			input: "cp file.txt ~/Desktop/",
			want:  []string{"cp", "file.txt", "~/Desktop/"},
		},
		{
			name:  "path with spaces in double quotes",
			input: `cp "cloud:/新建 Word 文档.docx" .`,
			want:  []string{"cp", "cloud:/新建 Word 文档.docx", "."},
		},
		{
			name:  "path with spaces in single quotes",
			input: `cp 'cloud:/新建 Word 文档.docx' .`,
			want:  []string{"cp", "cloud:/新建 Word 文档.docx", "."},
		},
		{
			name:  "escaped spaces with backslash",
			input: `cp cloud:/新建\ Word\ 文档.docx .`,
			want:  []string{"cp", "cloud:/新建 Word 文档.docx", "."},
		},
		{
			name:  "trailing space becomes empty last arg",
			input: "cp cloud:/dir/ ",
			want:  []string{"cp", "cloud:/dir/", ""},
		},
		{
			name:  "trailing space on second arg",
			input: "cp a.txt b.txt ",
			want:  []string{"cp", "a.txt", "b.txt", ""},
		},
		{
			name:  "extra whitespace between args",
			input: "cp   a.txt   b.txt",
			want:  []string{"cp", "a.txt", "b.txt"},
		},
		{
			name:  "only spaces",
			input: "   ",
			want:  nil,
		},
		{
			name:  "pipe not relevant to split",
			input: "ls | grep foo",
			want:  []string{"ls", "|", "grep", "foo"},
		},
		{
			name:  "mixed single and double quotes",
			input: `cp "file with spaces.txt" 'another file.txt'`,
			want:  []string{"cp", "file with spaces.txt", "another file.txt"},
		},
		{
			name:  "after tab complete then space",
			input: "cp cloud:/dir/ ",
			want:  []string{"cp", "cloud:/dir/", ""},
		},
		{
			name:  "after tab complete then space then more space",
			input: "cp cloud:/dir/   ",
			want:  []string{"cp", "cloud:/dir/", ""},
		},
		{
			name:  "after first tab complete then space ready for dest",
			input: "cp host:./src/ ",
			want:  []string{"cp", "host:./src/", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitArgs(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitArgs(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
