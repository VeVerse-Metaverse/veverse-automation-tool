package main

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"os"
)

func gitPublicKey() (publicKey *ssh.PublicKeys, err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return publicKey, fmt.Errorf("failed to get a public key: %v", err)
	}

	sshPath := homeDir + "/.ssh/veverse_builder_id_rsa"
	sshKey, _ := os.ReadFile(sshPath)

	publicKey, err = ssh.NewPublicKeys("git", sshKey, "")
	if err != nil {
		return publicKey, fmt.Errorf("failed to get a public key: %v", err)
	}

	return
}

func gitRepo(dir string) (*git.Repository, error) {
	r, err := git.PlainOpen(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo at %s: %v", dir, err)
	}
	return r, nil
}

func gitBranch(r *git.Repository) (string, error) {
	if r == nil {
		return "", fmt.Errorf("invalid repository")
	}

	h, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get a git worktree: %v", err)
	}

	return h.Name().Short(), nil
}

func gitCheckout(r *git.Repository, branch string) error {
	if r == nil {
		return fmt.Errorf("invalid repository")
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get a git worktree: %v", err)
	}

	err = w.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch), Force: true})
	if err != nil {
		return fmt.Errorf("failed to checkout a git repo branch %s: %v", branch, err)
	}

	return nil
}

func gitCheckoutCommit(r *git.Repository, hash plumbing.Hash) error {
	if r == nil {
		return fmt.Errorf("invalid repository")
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get a git worktree: %v", err)
	}

	err = w.Checkout(&git.CheckoutOptions{Hash: hash, Force: true})
	if err != nil {
		return fmt.Errorf("failed to checkout a git repo commit %s: %v", hash.String(), err)
	}

	return nil
}

func gitPull(r *git.Repository) error {
	if r == nil {
		return fmt.Errorf("invalid repository")
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get a git worktree: %v", err)
	}

	auth, err := gitPublicKey()
	if err != nil {
		return fmt.Errorf("failed to get a public key: %v", err)
	}

	if err = w.Pull(&git.PullOptions{RemoteName: "origin", Auth: auth, Force: true}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("failed to pull changes: %v", err)
		}
	}

	return nil
}

func projectLatestContentVersion() (string, error) {
	return "1.0.0", nil
}

func gitLatestTag(repository *git.Repository) (string, error) {
	tagRefs, err := repository.Tags()
	if err != nil {
		return "", err
	}

	var latestTagCommit *object.Commit
	var latestTagName string
	err = tagRefs.ForEach(func(tagRef *plumbing.Reference) error {
		revision := plumbing.Revision(tagRef.Name().String())
		tagCommitHash, err := repository.ResolveRevision(revision)
		if err != nil {
			return err
		}

		commit, err := repository.CommitObject(*tagCommitHash)
		if err != nil {
			return err
		}

		if latestTagCommit == nil {
			latestTagCommit = commit
			latestTagName = tagRef.Name().Short()
		}

		if commit.Committer.When.After(latestTagCommit.Committer.When) {
			latestTagCommit = commit
			latestTagName = tagRef.Name().Short()
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return latestTagName, nil
}

func gitTag(repository *git.Repository, tagName string) (*plumbing.Hash, error) {
	tagRefs, err := repository.Tags()
	if err != nil {
		return nil, err
	}

	var hash plumbing.Hash
	err = tagRefs.ForEach(func(tagRef *plumbing.Reference) error {
		if tagRef.Name().Short() == tagName {
			hash = tagRef.Hash()
			return nil
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &hash, nil
}
