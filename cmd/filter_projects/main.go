package main

import (
	"fmt"
	"log"

	configs "github.com/oleksandr-holan/gitlab-setup/pkg/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func main() {
	config, err := configs.NewGitLabConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	gitlabClient, err := gitlab.NewClient(config.AccessToken)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	projects, err := listGroupProjects(gitlabClient, config.MainGroupID)
	if err != nil {
		log.Fatalf("Failed to list projects: %v", err)
	}

	// Create two slices to store results
	hasDevBranchProjects := make([]string, 0)
	noDevBranchProjects := make([]string, 0)

	// Check each project and accumulate results
	for _, project := range projects {
		hasDevBranch, err := hasEnvironmentDevBranch(gitlabClient, project.ID)
		if err != nil {
			log.Printf("Error checking branches for project %s: %v", project.Name, err)
			continue
		}

		projectInfo := fmt.Sprintf("%s (ID: %d)", project.Name, project.ID)
		if hasDevBranch {
			hasDevBranchProjects = append(hasDevBranchProjects, projectInfo)
		} else {
			noDevBranchProjects = append(noDevBranchProjects, projectInfo)
		}
	}

	// Print results in groups
	fmt.Printf("\nProjects with environment/dev branch (%d):\n", len(hasDevBranchProjects))
	fmt.Println("----------------------------------------")
	for _, project := range hasDevBranchProjects {
		fmt.Printf("- %s\n", project)
	}

	fmt.Printf("\nProjects without environment/dev branch (%d):\n", len(noDevBranchProjects))
	fmt.Println("------------------------------------------")
	for _, project := range noDevBranchProjects {
		fmt.Printf("- %s\n", project)
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
