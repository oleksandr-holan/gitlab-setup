package testutil

import (
	"testing"

	configs "github.com/oleksandr-holan/gitlab-setup/pkg/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const (
	letterBytes  = "abcdefghijklmnopqrstuvwxyz"
	stringLength = 8
)

type GitLabTestHelper struct {
	Client *gitlab.Client
	Config *configs.GitLab
	T      *testing.T
}

func NewGitLabTestHelper(t *testing.T, config *configs.GitLab) *GitLabTestHelper {
	client, err := gitlab.NewClient(config.AccessToken, gitlab.WithBaseURL(config.GitlabURL))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}
	return &GitLabTestHelper{
		Client: client,
		Config: config,
		T:      t,
	}
}
