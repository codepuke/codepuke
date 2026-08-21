package main

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// resolveCommit resolves ref (branch, tag, or hash) in the repository at
// path to a full commit hash. Reading at a resolved commit makes sync output
// reproducible even when the sibling worktree is dirty.
func resolveCommit(path, ref string) (string, error) {
	out, err := exec.Command("git", "-C", path, "rev-parse", "--verify", "--quiet", ref+"^{commit}").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("resolve %q in %s: %s", ref, path, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("resolve %q in %s: %w", ref, path, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// forEachFile streams every tracked regular file at commit, via git archive,
// without touching the worktree.
func forEachFile(repoPath, commit string, fn func(name string, data []byte) error) error {
	cmd := exec.Command("git", "-C", repoPath, "archive", "--format=tar", commit)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("git archive %s: %w", repoPath, err)
	}

	tr := tar.NewReader(stdout)
	var walkErr error
	for walkErr == nil {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			walkErr = err
			break
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			walkErr = err
			break
		}
		walkErr = fn(hdr.Name, data)
	}

	if walkErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return walkErr
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git archive %s: %w: %s", repoPath, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
