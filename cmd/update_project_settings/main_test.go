package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	graphql "github.com/hasura/go-graphql-client"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/oauth2"

	"github.com/oleksandr-holan/gitlab-setup/internal/testutil"
	configs "github.com/oleksandr-holan/gitlab-setup/pkg/config"
	"github.com/oleksandr-holan/gitlab-setup/pkg/gitlabGraphQL"
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

	// Initialize GraphQL client
	graphqlClient := graphql.NewClient(
		fmt.Sprintf("%s/api/graphql", config.GitlabURL),
		&http.Client{
			Transport: &oauth2.Transport{
				Source: oauth2.StaticTokenSource(
					&oauth2.Token{AccessToken: config.AccessToken},
				),
			},
		},
	)

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
		if !project.AutocloseReferencedIssues {
			t.Errorf("Project %d should have autoclose_referenced_issues 'true', but got '%t'",
				project.ID, project.AutocloseReferencedIssues)
		}
		if project.MergeMethod != gitlab.FastForwardMerge {
			t.Errorf("Project %d should have merge_method 'ff', but got '%s'",
				project.ID, project.MergeMethod)
		}
		if project.ForkedFromProject != nil && !project.MergeRequestDefaultTargetSelf {
			t.Errorf("Project %d should have mr_default_target_self 'true', but got '%t'",
				project.ID, project.MergeRequestDefaultTargetSelf)
		}

		// Test GraphQL target branch rules
		rules, err := gitlabGraphQL.GetTargetBranchRules(context.Background(), graphqlClient, project.PathWithNamespace)
		if err != nil {
			t.Errorf("Failed to get target branch rules for project %d: %v", project.ID, err)
			continue
		}

		// Check if the expected rule exists
		foundRule := false
		for _, rule := range rules {
			if rule.Name == "*" && rule.TargetBranch == project.DefaultBranch {
				foundRule = true
				break
			}
		}

		if !foundRule {
			t.Errorf("Project %d should have a target branch rule with name '*' and target branch 'environment/dev'",
				project.ID)
		}
	}
}
