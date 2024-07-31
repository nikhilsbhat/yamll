package yamll

import (
	"fmt"
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
func (dependency *Dependency) Git(log *slog.Logger) (string, error) {
	gitMetaData, err := dependency.getGitMetaData()
	if err != nil {
		return "", err
	}

	cloneOptions := &git.CloneOptions{
		URL:      gitMetaData.gitBaseURL,
		Progress: os.Stdout,
	}

	if len(dependency.Auth.CaContent) != 0 {
		cloneOptions.CABundle = []byte(dependency.Auth.CaContent)
	}

	switch gitMetaData.ssh {
	case true:
		log.Debug("the git import is of type ssh, so setting ssh based auth")

		sshKEY, err := os.ReadFile(dependency.Auth.SSHKey)
		if err != nil {
			return "", &errors.YamllError{Message: fmt.Sprintf("reading ssh key '%s' errored with %v", dependency.Auth.SSHKey, err)}
		}

		signer, err := ssh.ParsePrivateKey(sshKEY)
		if err != nil {
			return "", err
		}

		cloneOptions.Auth = &gitssh.PublicKeys{User: "git", Signer: signer}

	case false:
		log.Debug("the git import is of type https, so setting http based auth")

		auth := &http.BasicAuth{
			Username: dependency.Auth.UserName,
			Password: dependency.Auth.Password,
		}

		if len(dependency.Auth.BarerToken) != 0 {
			auth.Password = dependency.Auth.BarerToken
		}

		cloneOptions.Auth = auth
	}

	tempDir := filepath.Join(os.TempDir(), "yamll_git"+uuid.New().String())
	if err = os.MkdirAll(tempDir, defaultDirPermissions); err != nil {
		return "", &errors.YamllError{Message: "failed to crete temp directory for cloning git material"}
	}

	log.Debug("cloning git repo", slog.String("repo", gitMetaData.gitBaseURL), slog.String("dir", tempDir))

	defer func(path string) {
		if err = os.RemoveAll(path); err != nil {
			log.Error(err.Error())
		}
	}(tempDir)

	repo, err := git.PlainClone(tempDir, false, cloneOptions)
	if err != nil {
		return "", err
	}

	if err = checkoutRevision(repo, gitMetaData.referenceName); err != nil {
		return "", err
	}

	gitFileContent, err := os.ReadFile(filepath.Join(tempDir, gitMetaData.path))
	if err != nil {
		return "", &errors.YamllError{Message: fmt.Sprintf("reading content from file of git errored with '%v'", err)}
	}

	return string(gitFileContent), nil
}

//nolint:gomnd
func (dependency *Dependency) getGitMetaData() (*gitMeta, error) {
	dependency.Path = strings.ReplaceAll(dependency.Path, "git+", "")

	isSSH := strings.HasPrefix(dependency.Path, "ssh://")

	var gitBaseURL, remainingPath string

	if isSSH {
		gitParsedURL := strings.SplitN(dependency.Path, "@", 3)
		if len(gitParsedURL) != 3 {
			return nil, &errors.YamllError{Message: fmt.Sprintf("unable to split git url '%s'", dependency.Path)}
		}

		gitBaseURL = fmt.Sprintf("git@%v", gitParsedURL[1])

		remainingPath = fmt.Sprintf("https://%v@%v", gitParsedURL[1], gitParsedURL[2])
	} else {
		gitParsedURL := strings.SplitN(dependency.Path, "@", 2)
		if len(gitParsedURL) != 2 {
			return nil, &errors.YamllError{Message: fmt.Sprintf("unable to parse git url '%s'", dependency.Path)}
		}

		gitBaseURL = gitParsedURL[0]

		remainingPath = dependency.Path
	}

	parsedRef := strings.SplitN(strings.SplitN(remainingPath, "?", 2)[0], "@", 2)
	if len(parsedRef) != 2 {
		return nil, &errors.YamllError{Message: fmt.Sprintf("unable to parse ref from '%s'", remainingPath)}
	}

	parsedPath := strings.SplitN(strings.SplitN(remainingPath, "?", 2)[1], "=", 2)
	if len(parsedPath) != 2 {
		return nil, &errors.YamllError{Message: fmt.Sprintf("unable to parse path from '%s'", remainingPath)}
	}

	return &gitMeta{
		gitBaseURL:    gitBaseURL,
		referenceName: parsedRef[1],
		path:          parsedPath[1],
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
