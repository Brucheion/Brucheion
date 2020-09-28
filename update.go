package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/tj/go-update"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/kierdavis/ansi"

	"github.com/coreos/go-semver/semver"
	"github.com/tj/go-update/progress"
	githubStore "github.com/tj/go-update/stores/github"
)

func releasesToString(releases []*update.Release) string {
	var val []string
	for _, r := range releases {
		val = append(val, r.Version)
	}

	return strings.Join(val, ", ")
}

// omitVersionPrefix omits a prefixed 'v' of a version string.
func omitVersionPrefix(v string) string {
	if v[0] == 'v' {
		return v[1:]
	}
	return v
}

// sortReleases sorts a slice of update.Release by their semantic versions from newest to oldest.
func sortReleases(releases []*update.Release) {
	sort.Slice(releases, func(i, j int) bool {
		a, _ := semver.NewVersion(omitVersionPrefix(releases[i].Version))
		b, _ := semver.NewVersion(omitVersionPrefix(releases[j].Version))

		return b.LessThan(*a)
	})
}

// transformRelease returns an update.Release.
func transformRelease(r *github.RepositoryRelease) *update.Release {
	out := &update.Release{
		Version:     r.GetTagName(),
		Notes:       r.GetBody(),
		PublishedAt: r.GetPublishedAt().Time,
		URL:         r.GetURL(),
	}

	for _, a := range r.Assets {
		out.Assets = append(out.Assets, &update.Asset{
			Name:      a.GetName(),
			Size:      a.GetSize(),
			URL:       a.GetBrowserDownloadURL(),
			Downloads: a.GetDownloadCount(),
		})
	}

	return out
}

// getNewerReleases returns all newer releases of a GitHub repository, sorted from newest to oldest.
func getNewerReleases(s *githubStore.Store, prerelease bool, timeout time.Duration) (release []*update.Release, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	gh := github.NewClient(nil)

	releases, _, err := gh.Repositories.ListReleases(ctx, s.Owner, s.Repo, nil)
	if err != nil {
		return nil, err
	}

	current, _ := semver.NewVersion(Version)
	var newer []*update.Release

	for _, r := range releases {
		if !prerelease && *r.Prerelease {
			continue
		}

		v, err := semver.NewVersion(omitVersionPrefix(r.GetTagName()))
		if err != nil || !current.LessThan(*v) {
			continue
		}

		newer = append(newer, transformRelease(r))
	}

	sortReleases(newer)
	return newer, nil
}

// getLatestRelease returns the latest newer release of a GitHub repository or nil.
func getLatestRelease(s *githubStore.Store, prerelease bool, timeout time.Duration) (release *update.Release, err error) {
	releases, err := getNewerReleases(s, prerelease, timeout)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return
	}
	return releases[0], nil
}

func installRelease(m *update.Manager, a *update.Asset) error {
	ansi.HideCursor()
	defer ansi.ShowCursor()

	tarball, err := a.DownloadProxy(progress.Reader)
	if err != nil {
		fmt.Printf("Error while downloading release: %s\n", err)
		return err
	}

	bin, err := os.Executable()
	if err != nil {
		fmt.Println("Error while obtaining executable path")
		return err
	}

	dir := path.Dir(bin)
	if err := m.InstallTo(tarball, dir); err != nil {
		fmt.Printf("Error while installing release: %s\n", err)
		return err
	}
	return nil
}

func handleUpdates() bool {
	store := &githubStore.Store{
		Owner:   "brucheion",
		Repo:    "brucheion",
		Version: "2.0.1",
	}
	m := &update.Manager{
		Command: "Brucheion",
		Store:   store,
	}

	latest, err := getLatestRelease(store, false, 5*time.Second)
	if err != nil {
		fmt.Printf("Error while retrieving latest release: #{err}\n")
		return false
	} else if latest == nil {
		fmt.Println("No newer release available.")
		return false
	}

	a := latest.FindTarball(runtime.GOOS, runtime.GOARCH)
	if a == nil {
		fmt.Println("There is a newer release available, but binaries for your system are not included.")
		return false
	}

	fmt.Printf("The latest release with version %s will be installed.\n\n", latest.Version)
	err = installRelease(m, a)
	if err != nil {
		return false
	}

	fmt.Printf("\n Updated to %s. Brucheion will exit now in order to apply the update.\n", latest.Version)
	return true
}
