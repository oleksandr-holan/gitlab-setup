package main

import (
	"fmt"
	"testing"

	"github.com/oleksandr-holan/gitlab-setup/internal/testutil"
	configs "github.com/oleksandr-holan/gitlab-setup/pkg/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var config *configs.GitLab
var helper *testutil.GitLabTestHelper
var gitlabClient *gitlab.Client

func TestUpdateProjectSettings(t *testing.T) {
	var err error
	config, err = configs.NewGitLabConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Initialize GitLab client
	gitlabClient, err = gitlab.NewClient(config.AccessToken, gitlab.WithBaseURL(config.GitlabURL))
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	helper = testutil.NewGitLabTestHelper(t, config)
	subgroups, projects := helper.CreateTestStructure()

	t.Cleanup(func() {
		helper.CleanupTestStructure(subgroups, projects)
	})

	updatedProjects, err := UpdateProjects(
		config.GitlabURL,
		config.AccessToken,
		fmt.Sprint(config.MainGroupID),
	)
	if err != nil {
		t.Fatalf("Failed to update projects: %v", err)
	}

	for _, project := range updatedProjects {
		if project.SquashOption != gitlab.SquashOptionDefaultOn {
			t.Errorf("Project %d should have squash_option 'default_on', but got '%s'",
				project.ID, project.SquashOption)
		}
		if project.DefaultBranch != "environment/dev" {
			t.Errorf("Project %d should have default_branch 'environment/dev', but got '%s'",
				project.ID, project.DefaultBranch)
		}
		if project.AutocloseReferencedIssues != true {
			t.Errorf("Project %d should have autoclose_referenced_issues 'true', but got '%s'",
				project.ID, project.DefaultBranch)
		}
		if project.MergeMethod != gitlab.FastForwardMerge {
			t.Errorf("Project %d should have merge_method 'ff', but got '%s'",
				project.ID, project.DefaultBranch)
		}
	}
}
