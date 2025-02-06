package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	MainGroupID int    `json:"main_group_id"`
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
	rand.Seed(time.Now().UnixNano())
}

func loadConfig() error {
	configPath := filepath.Join(".", "config.json")

	file, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config file: %v", err)
	}

	if err := json.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("parsing config file: %v", err)
	}

	if config.GitlabURL == "" || config.AccessToken == "" || config.MainGroupID == 0 {
		return fmt.Errorf("gitlab_url, access_token, and main_group_id must be set in config.json")
	}

	return nil
}

func TestUpdateSquashOption(t *testing.T) {
	subGroup1 := createGroup(t, config.MainGroupID, generateRandomString())
	subGroup2 := createGroup(t, subGroup1.ID, generateRandomString())
	subgroups := []GitLabGroup{
		subGroup1,
		subGroup2,
	}

	projects := []GitLabProject{
		createProject(t, subGroup2.ID, generateRandomString(), "never"),
		createProject(t, subGroup1.ID, generateRandomString(), "always"),
		createProject(t, config.MainGroupID, generateRandomString(), "default_off"),
	}

	t.Cleanup(func() {
		cleanupSubgroupsAndProjects(t, subgroups, projects)
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

func createGroup(t *testing.T, parentID int, name string) GitLabGroup {
	t.Helper()

	url := fmt.Sprintf("%s/api/v4/groups", config.GitlabURL)
	data := map[string]interface{}{
		"name": name,
		"path": name,
	}
	if parentID != 0 {
		data["parent_id"] = parentID
	}

	resp := makeRequest(t, "POST", url, data)
	defer resp.Body.Close()

	// Log the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	t.Logf("Response Body: %s", string(bodyBytes))
	t.Logf("Status Code: %d", resp.StatusCode)

	// Decode the response body
	var group GitLabGroup
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&group); err != nil {
		t.Fatalf("Failed to decode group response: %v", err)
	}

	return group
}

func createProject(t *testing.T, groupID int, name, squashOption string) GitLabProject {
	t.Helper()

	url := fmt.Sprintf("%s/api/v4/projects", config.GitlabURL)
	data := map[string]interface{}{
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

func cleanupSubgroupsAndProjects(t *testing.T, subgroups []GitLabGroup, projects []GitLabProject) {
	t.Helper()

	for _, project := range projects {
		url := fmt.Sprintf("%s/api/v4/projects/%d", config.GitlabURL, project.ID)
		resp := makeRequest(t, "DELETE", url, nil)
		resp.Body.Close()
	}

	for _, subgroup := range subgroups {
		url := fmt.Sprintf("%s/api/v4/groups/%d", config.GitlabURL, subgroup.ID)
		resp := makeRequest(t, "DELETE", url, nil)
		resp.Body.Close()
	}
}

func makeRequest(t *testing.T, method, url string, data map[string]interface{}) *http.Response {
	t.Helper()

	var req *http.Request
	var err error

	if data != nil {
		jsonData, marshalErr := json.Marshal(data)
		if marshalErr != nil {
			t.Fatalf("Failed to marshal request data: %v", marshalErr)
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
