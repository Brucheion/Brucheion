package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/tj/go-update"
	"log"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/kierdavis/ansi"

	"github.com/coreos/go-semver/semver"
	"github.com/tj/go-update/progress"
	githubReleases "github.com/tj/go-update/stores/github"
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
func getNewerReleases(
	store *githubReleases.Store,
	current *semver.Version,
	prerelease bool,
	timeout time.Duration,
) ([]*update.Release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	gh := github.NewClient(nil)

	releases, _, err := gh.Repositories.ListReleases(ctx, store.Owner, store.Repo, nil)
	if err != nil {
		return nil, err
	}

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
func getLatestRelease(
	store *githubReleases.Store,
	current *semver.Version,
	prerelease bool,
	timeout time.Duration,
) (*update.Release, error) {
	releases, err := getNewerReleases(store, current, prerelease, timeout)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil
	}
	return releases[0], nil
}

// installRelease replaces the current executable with a downloaded release binary.
func installRelease(m *update.Manager, a *update.Asset, dir string) error {
	ansi.HideCursor()
	defer ansi.ShowCursor()

	tarball, err := a.DownloadProxy(progress.Reader)
	if err != nil {
		return errors.Wrap(err, "Error while downloading release")
	}

	if err := m.InstallTo(tarball, dir); err != nil {
		return errors.Wrap(err, "Error while installing release")
	}
	return nil
}

// handleUpdates checks for updates and returns true if any were installed.
func handleUpdates() bool {
	current, err := semver.NewVersion(omitVersionPrefix(Version))
	if err != nil {
		log.Fatalf("Could not properly parse current version information %q.", Version)
	}

	command := "Brucheion"
	if runtime.GOOS == "windows" {
		command += ".exe"
	}

	store := &githubReleases.Store{
		Owner:   "brucheion",
		Repo:    "brucheion",
		Version: current.String(),
	}
	m := &update.Manager{
		Command: command,
		Store:   store,
	}

	latest, err := getLatestRelease(store, current, false, 5*time.Second)
	if err != nil {
		log.Printf("Error while retrieving update information: %s\n", err)
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

	fmt.Printf("There is a newer release with version %s available.\nDo you want to install it now? (y/[n]) ", latest.Version)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	choice := scanner.Text()
	fmt.Println()

	if strings.ToLower(choice) != "y" {
		fmt.Println("Update will not be installed now.")
		return false
	}

	bin, err := os.Executable()
	if err != nil {
		log.Fatalf("Error while obtaining executable path: %s\n", err)
	}
	dir := path.Dir(bin)

	fmt.Printf("The latest release with version %s will be installed.\n", latest.Version)
	err = installRelease(m, a, dir)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println()

	ex := path.Base(bin)
	if ex != command {
		fmt.Printf("Installed release %s to %q.\n", latest.Version, path.Join(dir, command))
	} else {
		fmt.Printf("Updated to %s.\n", latest.Version)
	}
	fmt.Println("Brucheion will exit now in order to apply the update.")
	return true
}
