package testutil

import (
	configs "github.com/oleksandr-holan/gitlab-setup/pkg/config"
	"testing"
)

const (
	letterBytes  = "abcdefghijklmnopqrstuvwxyz"
	stringLength = 8
)

type GitLabTestHelper struct {
	Config *configs.GitLab
	T      *testing.T
}

func NewGitLabTestHelper(t *testing.T, config *configs.GitLab) *GitLabTestHelper {
	return &GitLabTestHelper{
		Config: config,
		T:      t,
	}
}
