package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatalf("Usage: %s <gitlab-url> <access-token> <group-id>", os.Args[0])
	}

	gitlabURL := os.Args[1]
	accessToken := os.Args[2]
	groupID := os.Args[3]

	gid, err := strconv.Atoi(groupID)
	if err != nil {
		log.Fatalf("Invalid group ID: %v", err)
	}

	client := &http.Client{}

	// Get all projects in the group and its subgroups
	projects, err := getAllProjectsInGroup(client, gitlabURL, accessToken, gid)
	if err != nil {
		log.Fatalf("Error fetching projects: %v", err)
	}

	// Update the squash option for each project
	for _, project := range projects {
		err := updateProjectSquashOption(client, gitlabURL, accessToken, project.ID)
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

// updateProjectSquashOption updates the squash option for a project
func updateProjectSquashOption(client *http.Client, gitlabURL, accessToken string, projectID int) error {
	url := fmt.Sprintf("%s/api/v4/projects/%d", gitlabURL, projectID)

	data := map[string]string{
		"squash_option": "default_on", // Options: "never", "always", "default_on", "default_off"
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling data: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("creating request: %v", err)
	}
	req.Header.Add("PRIVATE-TOKEN", accessToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Project represents a GitLab project
type Project struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
}
