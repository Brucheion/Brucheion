package main

import (
	"github.com/tj/go-update"
	"testing"
)

func TestOmitVersionPrefix(t *testing.T) {
	fixtures := []struct {
		x string
		y string
	}{
		{"v1.0.0", "1.0.0"},
		{"voobar", "oobar"},
		{"foobar", "foobar"},
	}

	for _, f := range fixtures {
		if omitVersionPrefix(f.x) != f.y {
			t.Errorf("Did not emit version prefix: %s (input), %s (output)\n", f.x, f.y)
		}
	}
}

func equalReleases(a []*update.Release, b []*update.Release) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Version != b[i].Version {
			return false
		}
	}

	return true
}

func TestSortReleases(t *testing.T) {
	a := &update.Release{Version: "v1.0.0"}
	b := &update.Release{Version: "v1.1.0"}
	c := &update.Release{Version: "v0.9.0"}
	d := &update.Release{Version: "v1.0.0-beta.1"}

	fixture := []*update.Release{a, d, b, c}
	expected := []*update.Release{b, a, d, c}

	sortReleases(fixture)

	if !equalReleases(fixture, expected) {
		t.Errorf("Sorted releases do not match.\nResult: %s\nExpected: %s\n",
			releasesToString(fixture), releasesToString(expected))
	}
}
