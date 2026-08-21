package anchor

import (
	"context"
	"errors"
	"strings"
)

// Commit is a commit whose message carries a matching direction trailer.
type Commit struct {
	SHA  string
	Date string // committer date, ISO 8601
}

// CommitsWithTrailer lists commits reachable from HEAD (newest first) whose
// Festival-Direction trailer equals hash, up to limit (0 = no limit).
func CommitsWithTrailer(ctx context.Context, repo Repo, hash string, limit int) ([]Commit, error) {
	if _, err := repo.Head(ctx); errors.Is(err, ErrNoCommits) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	// Records separated by \x1e, fields by \x1f: sha, committer date, body.
	out, err := runGit(ctx, repo.Root, "log", "--format=%H%x1f%cI%x1f%B%x1e", "HEAD")
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, rec := range strings.Split(out, "\x1e") {
		fields := strings.SplitN(strings.TrimLeft(rec, "\n"), "\x1f", 3)
		if len(fields) != 3 {
			continue
		}
		tr, err := ParseTrailers(fields[2])
		if err != nil || tr.DirectionHash != hash {
			continue
		}
		commits = append(commits, Commit{SHA: fields[0], Date: fields[1]})
		if limit > 0 && len(commits) >= limit {
			break
		}
	}
	return commits, nil
}
