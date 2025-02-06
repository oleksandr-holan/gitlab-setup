package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	gitlabURL    = "http://your-gitlab-instance.com"
	accessToken  = "your-access-token"
	letterBytes  = "abcdefghijklmnopqrstuvwxyz"
	stringLength = 8
)

type GitLabGroup struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type GitLabProject struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	SquashOption string `json:"squash_option"`
}

func TestUpdateSquashOption(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	// Create main group
	mainGroup := createGroup(t, "", generateRandomString())

	// Create first subgroup
	subGroup1 := createGroup(t, fmt.Sprint(mainGroup.ID), generateRandomString())

	// Create second subgroup under first subgroup
	subGroup2 := createGroup(t, fmt.Sprint(subGroup1.ID), generateRandomString())

	// Create projects with different squash options
	projects := []GitLabProject{
		createProject(t, fmt.Sprint(subGroup2.ID), generateRandomString(), "never"),      // Deepest level
		createProject(t, fmt.Sprint(subGroup1.ID), generateRandomString(), "always"),     // Mid level
		createProject(t, fmt.Sprint(mainGroup.ID), generateRandomString(), "default_on"), // Top level
	}

	// Run the main program
	os.Args = []string{"cmd", gitlabURL, accessToken, fmt.Sprint(mainGroup.ID)}
	main()

	// Verify all projects now have squash_option set to default_on
	for _, project := range projects {
		currentSquashOption := getProjectSquashOption(t, project.ID)
		if currentSquashOption != "default_on" {
			t.Errorf("Project %d should have squash_option 'default_on', but got '%s'",
				project.ID, currentSquashOption)
		}
	}

	// Cleanup
	cleanupGroup(t, mainGroup.ID)
}

func createGroup(t *testing.T, parentID, name string) GitLabGroup {
	t.Helper()

	url := fmt.Sprintf("%s/api/v4/groups", gitlabURL)
	data := map[string]string{
		"name": name,
		"path": name,
	}
	if parentID != "" {
		data["parent_id"] = parentID
	}

	resp := makeRequest(t, "POST", url, data)
	defer resp.Body.Close()

	var group GitLabGroup
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		t.Fatalf("Failed to decode group response: %v", err)
	}

	return group
}

func createProject(t *testing.T, groupID, name, squashOption string) GitLabProject {
	t.Helper()

	url := fmt.Sprintf("%s/api/v4/projects", gitlabURL)
	data := map[string]string{
		"name":          name,
		"path":          name,
		"namespace_id":  groupID,
		"squash_option": squashOption,
	}

	resp := makeRequest(t, "POST", url, data)
	defer resp.Body.Close()

	var project GitLabProject
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		t.Fatalf("Failed to decode project response: %v", err)
	}

	return project
}

func getProjectSquashOption(t *testing.T, projectID int) string {
	t.Helper()

	url := fmt.Sprintf("%s/api/v4/projects/%d", gitlabURL, projectID)
	resp := makeRequest(t, "GET", url, nil)
	defer resp.Body.Close()

	var project GitLabProject
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		t.Fatalf("Failed to decode project response: %v", err)
	}

	return project.SquashOption
}

func cleanupGroup(t *testing.T, groupID int) {
	t.Helper()

	url := fmt.Sprintf("%s/api/v4/groups/%d", gitlabURL, groupID)
	resp := makeRequest(t, "DELETE", url, nil)
	resp.Body.Close()
}

func makeRequest(t *testing.T, method, url string, data map[string]string) *http.Response {
	t.Helper()

	var req *http.Request
	var err error

	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("Failed to marshal request data: %v", err)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Add("PRIVATE-TOKEN", accessToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("Request failed with status code: %d", resp.StatusCode)
	}

	return resp
}

func generateRandomString() string {
	b := make([]byte, stringLength)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
