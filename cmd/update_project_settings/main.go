package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatalf("Usage: %s <gitlab-url> <access-token> <group-id>", os.Args[0])
	}

	gitlabURL := os.Args[1]
	accessToken := os.Args[2]
	groupID := os.Args[3]

	if !strings.HasPrefix(gitlabURL, "https://") {
		log.Fatalln("gitlab_url must use HTTPS (e.g., https://gitlab.example.com)")
	}
	gid, err := strconv.Atoi(groupID)
	if err != nil {
		log.Fatalf("Invalid group ID: %v", err)
	}

	client := &http.Client{}
	gitlabClient, err := gitlab.NewClient(accessToken, gitlab.WithBaseURL(gitlabURL))

	if err != nil {
		log.Fatalf("Failed to create GitLab client: %v", err)
	}

	// Get all projects in the group and its subgroups
	projects, err := getAllProjectsInGroup(client, gitlabURL, accessToken, gid)
	if err != nil {
		log.Fatalf("Error fetching projects: %v", err)
	}

	for _, project := range projects {
		err = updateProjectSettings(gitlabClient, project.ID)
		if err != nil {
			log.Printf("Error updating project %d: %v", project.ID, err)
		} else {
			log.Printf("Successfully updated project %d (%s)", project.ID, project.PathWithNamespace)
		}
	}
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

func updateProjectSettings(client *gitlab.Client, projectID int) error {
	options := &gitlab.EditProjectOptions{
		SquashOption: gitlab.Ptr(gitlab.SquashOptionValue("default_on")), // Options: "never", "always", "default_on", "default_off"
	}

	_, _, err := client.Projects.EditProject(projectID, options)
	if err != nil {
		return fmt.Errorf("updating project settings: %v", err)
	}

	return nil
}

// Project represents a GitLab project
type Project struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
}
