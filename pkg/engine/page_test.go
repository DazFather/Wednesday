package engine

import (
	"path/filepath"
	"testing"
)

func TestAssetURL(t *testing.T) {
	for _, test := range []struct {
		location string
		asset    string
		want     string
	}{
		{"", "style/app.css", "./style/app.css"},
		{"index.html", "style/app.css", "./style/app.css"},
		{"page.html", "script/app.js", "./script/app.js"},
		{"./page.html", "style/app.css", "./style/app.css"},
		{"sub/../page.html", "style/app.css", "./style/app.css"},
		{"sub/page.html", "style/app.css", "../style/app.css"},
		{"sub/dir/page.html", "script/app.js", "../../script/app.js"},
		{filepath.Join("sub", "dir", "page.html"), "style/app.css", "../../style/app.css"},
	} {
		p := page{Location: test.location}
		if got := p.assetURL(test.asset); got != test.want {
			t.Errorf("assetURL(%q, %q) = %q, want %q", test.location, test.asset, got, test.want)
		}
	}
}
