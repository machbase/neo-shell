package rels_test

import (
	"testing"

	"github.com/machbase/neo-shell/rels"
)

func TestFetch(t *testing.T) {
	rels.FetchGithubReleases("machbase", "machbase-neo")
}
