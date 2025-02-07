package main

import (
	"context"
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

func UpdateProjects(gitlabURL, accessToken, groupID string) ([]*gitlab.Project, error) {
	if !strings.HasPrefix(gitlabURL, "https://") {
		return nil, fmt.Errorf("gitlab_url must use HTTPS (e.g., https://gitlab.example.com)")
	}

	gid, err := strconv.Atoi(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %v", err)
	}

	// client := &http.Client{}
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
	projects, err := getAllProjectsInGroup(gitlabClient, gid)
	if err != nil {
		return nil, fmt.Errorf("error fetching projects: %v", err)
	}

	updatedProjects := make([]*gitlab.Project, 0, len(projects))
	for _, project := range projects {
		options := &gitlab.EditProjectOptions{
			SquashOption:              gitlab.Ptr(gitlab.SquashOptionDefaultOn),
			AutocloseReferencedIssues: gitlab.Ptr(true),
			MergeMethod:               gitlab.Ptr(gitlab.FastForwardMerge),
		}

		if project.ForkedFromProject != nil {
			options.MergeRequestDefaultTargetSelf = gitlab.Ptr(true)
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

func getAllProjectsInGroup(client *gitlab.Client, groupID int) ([]*gitlab.Project, error) {
	opt := &gitlab.ListGroupProjectsOptions{
		IncludeSubGroups: gitlab.Ptr(true),
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	var allProjects []*gitlab.Project
	for {
		projects, resp, err := client.Groups.ListGroupProjects(groupID, opt)
		if err != nil {
			return nil, fmt.Errorf("listing group projects: %v", err)
		}

		allProjects = append(allProjects, projects...)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		// Update the page number to get the next page
		opt.Page = resp.NextPage
	}

	return allProjects, nil
}
