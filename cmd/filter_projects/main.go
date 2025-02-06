package main

import (
	"fmt"
	"log"

	configs "github.com/oleksandr-holan/gitlab-setup/pkg/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func main() {
	// Replace with your GitLab token and group ID

	config, err := configs.NewGitLabConfig()

	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create a new GitLab client
	gitlabClient, err := gitlab.NewClient(config.AccessToken)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Recursively list all projects in the group
	projects, err := listGroupProjects(gitlabClient, config.MainGroupID)
	if err != nil {
		log.Fatalf("Failed to list projects: %v", err)
	}

	// Check each project for the environment/dev branch
	for _, project := range projects {
		hasDevBranch, err := hasEnvironmentDevBranch(gitlabClient, project.ID)
		if err != nil {
			log.Printf("Error checking branches for project %s: %v", project.Name, err)
			continue
		}

		if !hasDevBranch {
			fmt.Printf("Project %s (ID: %d) does not have an environment/dev branch\n", project.Name, project.ID)
		} else {
			fmt.Printf("Project %s (ID: %d) does has an environment/dev branch\n", project.Name, project.ID)
		}
	}
}

// listGroupProjects recursively lists all projects in a GitLab group
func listGroupProjects(gitlabClient *gitlab.Client, groupID interface{}) ([]*gitlab.Project, error) {
	var allProjects []*gitlab.Project

	opt := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		IncludeSubGroups: gitlab.Ptr(true),
	}

	for {
		projects, resp, err := gitlabClient.Groups.ListGroupProjects(groupID, opt)
		if err != nil {
			return nil, err
		}

		allProjects = append(allProjects, projects...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}

	return allProjects, nil
}

// hasEnvironmentDevBranch checks if a project has an environment/dev branch
func hasEnvironmentDevBranch(gitlabClient *gitlab.Client, projectID int) (bool, error) {
	branches, _, err := gitlabClient.Branches.ListBranches(projectID, &gitlab.ListBranchesOptions{})
	if err != nil {
		return false, err
	}

	for _, branch := range branches {
		if branch.Name == "environment/dev" {
			return true, nil
		}
	}

	return false, nil
}
