package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	letterBytes  = "abcdefghijklmnopqrstuvwxyz"
	stringLength = 8
)

type Config struct {
	GitlabURL   string `json:"gitlab_url"`
	AccessToken string `json:"access_token"`
}

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

var config Config

func init() {
	if err := loadConfig(); err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
}

func loadConfig() error {
	// Look for config.json in the same directory as the test file
	configPath := filepath.Join(".", "config.json")

	file, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config file: %v", err)
	}

	if err := json.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("parsing config file: %v", err)
	}

	if config.GitlabURL == "" || config.AccessToken == "" {
		return fmt.Errorf("gitlab_url and access_token must be set in config.json")
	}

	return nil
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
		createProject(t, fmt.Sprint(subGroup2.ID), generateRandomString(), "never"),       // Deepest level
		createProject(t, fmt.Sprint(subGroup1.ID), generateRandomString(), "always"),      // Mid level
		createProject(t, fmt.Sprint(mainGroup.ID), generateRandomString(), "default_off"), // Top level
	}

	// Run the main program
	os.Args = []string{"cmd", config.GitlabURL, config.AccessToken, fmt.Sprint(mainGroup.ID)}
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

	url := fmt.Sprintf("%s/api/v4/groups", config.GitlabURL)
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

	url := fmt.Sprintf("%s/api/v4/projects", config.GitlabURL)
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

	url := fmt.Sprintf("%s/api/v4/projects/%d", config.GitlabURL, projectID)
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

	url := fmt.Sprintf("%s/api/v4/groups/%d", config.GitlabURL, groupID)
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

	req.Header.Add("PRIVATE-TOKEN", config.AccessToken)
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
