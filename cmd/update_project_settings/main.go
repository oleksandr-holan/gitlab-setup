package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	graphql "github.com/hasura/go-graphql-client"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"golang.org/x/oauth2"

	"github.com/oleksandr-holan/gitlab-setup/pkg/gitlabGraphQL"
)

// Project represents a GitLab project
type Project struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
}

func UpdateProjects(gitlabURL, accessToken, groupID string) ([]*gitlab.Project, error) {
	if !strings.HasPrefix(gitlabURL, "https://") {
		return nil, fmt.Errorf("gitlab_url must use HTTPS (e.g., https://gitlab.example.com)")
	}

	gid, err := strconv.Atoi(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %v", err)
	}

	client := &http.Client{}
	gitlabClient, err := gitlab.NewClient(accessToken, gitlab.WithBaseURL(gitlabURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %v", err)
	}

	// Initialize GraphQL client
	graphqlClient := graphql.NewClient(
		fmt.Sprintf("%s/api/graphql", gitlabURL),
		&http.Client{
			Transport: &oauth2.Transport{
				Source: oauth2.StaticTokenSource(
					&oauth2.Token{AccessToken: accessToken},
				),
			},
		},
	)

	// Get all projects in the group and its subgroups
	projects, err := getAllProjectsInGroup(client, gitlabURL, accessToken, gid)
	if err != nil {
		return nil, fmt.Errorf("error fetching projects: %v", err)
	}

	updatedProjects := make([]*gitlab.Project, 0, len(projects))
	for _, project := range projects {
		options := &gitlab.EditProjectOptions{
			SquashOption:              gitlab.Ptr(gitlab.SquashOptionDefaultOn), // Options: "never", "always", "default_on", "default_off"
			AutocloseReferencedIssues: gitlab.Ptr(true),
			MergeMethod:               gitlab.Ptr(gitlab.FastForwardMerge),
		}

		if branch, _, err := gitlabClient.Branches.GetBranch(project.ID, "environment/dev"); err == nil && branch != nil {
			options.DefaultBranch = gitlab.Ptr("environment/dev")
		}

		updatedProject, _, err := gitlabClient.Projects.EditProject(project.ID, options)
		if err != nil {
			log.Printf("Error updating project %d: %v", project.ID, err)
			continue
		}

		// Create target branch rule using GraphQL
		err = gitlabGraphQL.CreateTargetBranchRule(context.Background(), graphqlClient, project.ID, "*", updatedProject.DefaultBranch)
		if err != nil {
			log.Printf("Error creating target branch rule for project %d: %v", project.ID, err)
		}

		updatedProjects = append(updatedProjects, updatedProject)
	}

	return updatedProjects, nil
}

// getAllProjectsInGroup fetches all projects in a group and its subgroups
func getAllProjectsInGroup(client *http.Client, gitlabURL, accessToken string, groupID int) ([]Project, error) {
	var allProjects []Project
	page := 1

	for {
		url := fmt.Sprintf("%s/api/v4/groups/%d/projects?include_subgroups=true&page=%d&per_page=100", gitlabURL, groupID, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %v", err)
		}
		req.Header.Add("PRIVATE-TOKEN", accessToken)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching projects: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var projects []Project
		if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
			return nil, fmt.Errorf("decoding response: %v", err)
		}

		allProjects = append(allProjects, projects...)

		nextPage := resp.Header.Get("X-Next-Page")
		if nextPage == "" || nextPage == "0" {
			break
		}
		page++
	}

	return allProjects, nil
}
