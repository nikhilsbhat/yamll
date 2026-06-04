package yamll

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/uuid"
	"github.com/nikhilsbhat/yamll/pkg/errors"
	"golang.org/x/crypto/ssh"
)

type gitMeta struct {
	gitBaseURL    string
	referenceName string
	path          string
	ssh           bool
}

// Git reads the data from the Git import.
func (dependency *Dependency) Git(log *slog.Logger) (File, error) {
	gitMetaData, err := dependency.getGitMetaData()
	if err != nil {
		return File{}, err
	}

	var depAuth Auth
	if dependency.Auth != nil {
		depAuth = *dependency.Auth
	}

	cloneOptions := &git.CloneOptions{
		URL:      gitMetaData.gitBaseURL,
		Progress: gitCloneProgressWriter(log),
	}

	if len(depAuth.CaContent) != 0 {
		cloneOptions.CABundle = []byte(depAuth.CaContent)
	}

	switch gitMetaData.ssh {
	case true:
		log.Debug("the git import is of type ssh, so setting ssh based auth")

		sshKEY, err := os.ReadFile(depAuth.SSHKey)
		if err != nil {
			return File{}, &errors.YamllError{Message: fmt.Sprintf("reading ssh key '%s' errored with %v", depAuth.SSHKey, err)}
		}

		signer, err := ssh.ParsePrivateKey(sshKEY)
		if err != nil {
			return File{}, err
		}

		cloneOptions.Auth = &gitssh.PublicKeys{User: "git", Signer: signer}

	case false:
		log.Debug("the git import is of type https, so setting http based auth")

		auth := &http.BasicAuth{
			Username: depAuth.UserName,
			Password: depAuth.Password,
		}

		if len(depAuth.BarerToken) != 0 {
			auth.Password = depAuth.BarerToken
		}

		cloneOptions.Auth = auth
	}

	tempDir := filepath.Join(os.TempDir(), "yamll_git"+uuid.New().String())
	if err = os.MkdirAll(tempDir, defaultDirPermissions); err != nil {
		return File{}, &errors.YamllError{Message: "failed to crete temp directory for cloning git material"}
	}

	log.Debug("cloning git repo", slog.String("repo", gitMetaData.gitBaseURL), slog.String("dir", tempDir))

	defer func(path string) {
		if err = os.RemoveAll(path); err != nil {
			log.Error(err.Error())
		}
	}(tempDir)

	repo, err := git.PlainClone(tempDir, false, cloneOptions)
	if err != nil {
		return File{}, err
	}

	if err = checkoutRevision(repo, gitMetaData.referenceName); err != nil {
		return File{}, err
	}

	yamlFilePath := filepath.Join(tempDir, gitMetaData.path)

	gitFileContent, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return File{}, &errors.YamllError{Message: fmt.Sprintf("reading content from file of git errored with '%v'", err)}
	}

	head, err := repo.Head()
	if err != nil {
		return File{}, err
	}

	sum := sha256.Sum256(gitFileContent)

	return File{
		Name: yamlFilePath,
		Data: string(gitFileContent),
		Meta: FileMeta{
			SHA256:    hex.EncodeToString(sum[:]),
			GitCommit: head.Hash().String(),
		},
	}, nil
}

func gitCloneProgressWriter(log *slog.Logger) io.Writer {
	if log != nil && log.Enabled(context.Background(), slog.LevelDebug) {
		return os.Stdout
	}

	return nil
}

//nolint:gomnd
func (dependency *Dependency) getGitMetaData() (*gitMeta, error) {
	dependency.Path = strings.ReplaceAll(dependency.Path, "git+", "")

	isSSH := strings.HasPrefix(dependency.Path, "ssh://")

	var gitBaseURL, remainingPath string

	if isSSH {
		afterScheme := strings.TrimPrefix(dependency.Path, "ssh://")
		userHost, sshPath, ok := strings.Cut(afterScheme, "@") //nolint:varnamelen

		if !ok || userHost == "" || sshPath == "" {
			return nil, &errors.YamllError{Message: fmt.Sprintf("unable to split git url '%s'", dependency.Path)}
		}

		refHost, refPath, ok := strings.Cut(sshPath, "@")
		if !ok || refHost == "" || refPath == "" {
			return nil, &errors.YamllError{Message: fmt.Sprintf("unable to split git url '%s'", dependency.Path)}
		}

		gitBaseURL = "git@" + userHost

		remainingPath = "https://" + userHost + "@" + refPath
	} else {
		baseURL, _, ok := strings.Cut(dependency.Path, "@") //nolint:varnamelen
		if !ok || baseURL == "" {
			return nil, &errors.YamllError{Message: fmt.Sprintf("unable to parse git url '%s'", dependency.Path)}
		}

		gitBaseURL = baseURL

		remainingPath = dependency.Path
	}

	refPath, query, ok := strings.Cut(remainingPath, "?") //nolint:varnamelen
	if !ok || query == "" {
		return nil, &errors.YamllError{Message: fmt.Sprintf("unable to parse path from '%s'", remainingPath)}
	}

	_, referenceName, ok := strings.Cut(refPath, "@") //nolint:varnamelen
	if !ok || referenceName == "" {
		return nil, &errors.YamllError{Message: fmt.Sprintf("unable to parse ref from '%s'", remainingPath)}
	}

	key, pathValue, ok := strings.Cut(query, "=") //nolint:varnamelen
	if !ok || key == "" || pathValue == "" {
		return nil, &errors.YamllError{Message: fmt.Sprintf("unable to parse path from '%s'", remainingPath)}
	}

	return &gitMeta{
		gitBaseURL:    gitBaseURL,
		referenceName: referenceName,
		path:          pathValue,
		ssh:           isSSH,
	}, nil
}

func checkoutRevision(repo *git.Repository, referenceName string) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(referenceName))
	if err != nil {
		return err
	}

	return worktree.Checkout(&git.CheckoutOptions{Hash: *hash})
}
