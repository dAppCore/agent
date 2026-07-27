// SPDX-License-Identifier: EUPL-1.2

package flow

import (
	"testing"

	core "dappco.re/go"
)

// TestFlow_MarkdownFrontMatter_Good — a fenced front-matter block returns its
// inner text (and CRLF endings normalise before parsing).
func TestFlow_MarkdownFrontMatter_Good(t *testing.T) {
	body, ok := markdownFrontMatter([]byte("---\ntitle: hi\nx: 1\n---\nbody text"))
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "title: hi\nx: 1", body)

	crlf, ok := markdownFrontMatter([]byte("---\r\nkey: val\r\n---\r\nbody"))
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "key: val", crlf)
}

// TestFlow_MarkdownFrontMatter_Bad — no opening fence, and an opening fence with
// no closing fence, both report "not front matter".
func TestFlow_MarkdownFrontMatter_Bad(t *testing.T) {
	if _, ok := markdownFrontMatter([]byte("just a plain document\n")); ok {
		t.Fatal("plain document must not parse as front matter")
	}
	if _, ok := markdownFrontMatter([]byte("---\nkey: val\nno closing fence")); ok {
		t.Fatal("an unterminated fence must not parse as front matter")
	}
}
