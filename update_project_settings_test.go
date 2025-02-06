package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/oleksandr-holan/gitlab-setup/internal/testutil"
	configs "github.com/oleksandr-holan/gitlab-setup/pkg/config"
	"github.com/oleksandr-holan/gitlab-setup/pkg/models"
)

var config *configs.GitLab
var helper *testutil.GitLabTestHelper

func TestUpdateSquashOption(t *testing.T) {
	var err error
	config, err = configs.NewGitLabConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	helper = testutil.NewGitLabTestHelper(t, config)
	subgroups, projects := helper.CreateTestStructure()

	t.Cleanup(func() {
		helper.CleanupTestStructure(subgroups, projects)
	})

	os.Args = []string{"cmd", config.GitlabURL, config.AccessToken, fmt.Sprint(config.MainGroupID)}
	main()

	for _, project := range projects {
		currentSquashOption := getProjectSquashOption(t, project.ID)
		if currentSquashOption != "default_on" {
			t.Errorf("Project %d should have squash_option 'default_on', but got '%s'",
				project.ID, currentSquashOption)
		}
	}
}

func getProjectSquashOption(t *testing.T, projectID int) string {
	t.Helper()

	url := fmt.Sprintf("%s/api/v4/projects/%d", config.GitlabURL, projectID)
	resp := helper.MakeRequest("GET", url, nil)
	defer resp.Body.Close()

	var project models.GitLabProject
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		t.Fatalf("Failed to decode project response: %v", err)
	}

	return project.SquashOption
}
