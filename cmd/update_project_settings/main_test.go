package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/oleksandr-holan/gitlab-setup/internal/testutil"
	configs "github.com/oleksandr-holan/gitlab-setup/pkg/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var config *configs.GitLab
var helper *testutil.GitLabTestHelper
var gitlabClient *gitlab.Client

func TestUpdateSquashOption(t *testing.T) {
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

	os.Args = []string{"cmd", config.GitlabURL, config.AccessToken, fmt.Sprint(config.MainGroupID)}
	main()

	// Refresh projects data after main() execution
	for i, project := range projects {
		updatedProject, _, err := gitlabClient.Projects.GetProject(project.ID, nil)
		if err != nil {
			t.Fatalf("Failed to get updated project %d: %v", project.ID, err)
		}
		projects[i] = updatedProject
	}

	for _, project := range projects {
		if project.SquashOption != "default_on" {
			t.Errorf("Project %d should have squash_option 'default_on', but got '%s'",
				project.ID, project.SquashOption)
		}
	}
}
