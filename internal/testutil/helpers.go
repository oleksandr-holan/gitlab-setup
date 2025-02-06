package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/oleksandr-holan/gitlab-setup/pkg/models"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (h *GitLabTestHelper) CreateTestStructure() ([]models.GitLabGroup, []models.GitLabProject) {
	subGroup1 := h.CreateGroup(h.Config.MainGroupID, h.GenerateRandomString())
	subGroup2 := h.CreateGroup(subGroup1.ID, h.GenerateRandomString())
	subgroups := []models.GitLabGroup{
		subGroup1,
		subGroup2,
	}

	projects := []models.GitLabProject{
		h.CreateProject(subGroup2.ID, h.GenerateRandomString(), "never"),
		h.CreateProject(subGroup1.ID, h.GenerateRandomString(), "always"),
		h.CreateProject(h.Config.MainGroupID, h.GenerateRandomString(), "default_off"),
	}

	return subgroups, projects
}

func (h *GitLabTestHelper) CreateGroup(parentID int, name string) models.GitLabGroup {
	h.T.Helper()

	url := fmt.Sprintf("%s/api/v4/groups", h.Config.GitlabURL)
	data := map[string]interface{}{
		"name": name,
		"path": name,
	}
	if parentID != 0 {
		data["parent_id"] = parentID
	}

	resp := h.MakeRequest("POST", url, data)
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		h.T.Fatalf("Failed to read response body: %v", err)
	}

	var group models.GitLabGroup
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&group); err != nil {
		h.T.Fatalf("Failed to decode group response: %v", err)
	}

	return group
}

func (h *GitLabTestHelper) CreateProject(groupID int, name, squashOption string) models.GitLabProject {
	h.T.Helper()

	url := fmt.Sprintf("%s/api/v4/projects", h.Config.GitlabURL)
	data := map[string]interface{}{
		"name":          name,
		"path":          name,
		"namespace_id":  groupID,
		"squash_option": squashOption,
	}

	resp := h.MakeRequest("POST", url, data)
	defer resp.Body.Close()

	var project models.GitLabProject
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		h.T.Fatalf("Failed to decode project response: %v", err)
	}

	return project
}

func (h *GitLabTestHelper) CleanupTestStructure(subgroups []models.GitLabGroup, projects []models.GitLabProject) {
	h.T.Helper()
	h.CleanupSubgroupsAndProjects(subgroups, projects)
}

func (h *GitLabTestHelper) MakeRequest(method, url string, data map[string]interface{}) *http.Response {
	h.T.Helper()

	var req *http.Request
	var err error

	if data != nil {
		jsonData, marshalErr := json.Marshal(data)
		if marshalErr != nil {
			h.T.Fatalf("Failed to marshal request data: %v", marshalErr)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		h.T.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Add("PRIVATE-TOKEN", h.Config.AccessToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		h.T.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		h.T.Fatalf("Request failed with status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp
}

func (h *GitLabTestHelper) GenerateRandomString() string {
	b := make([]byte, stringLength)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func (h *GitLabTestHelper) CleanupSubgroupsAndProjects(subgroups []models.GitLabGroup, projects []models.GitLabProject) {
	h.T.Helper()

	// Delete projects first to avoid dependencies
	for _, project := range projects {
		url := fmt.Sprintf("%s/api/v4/projects/%d", h.Config.GitlabURL, project.ID)
		resp := h.MakeRequest("DELETE", url, nil)
		resp.Body.Close()
	}

	for _, subgroup := range subgroups {
		url := fmt.Sprintf("%s/api/v4/groups/%d", h.Config.GitlabURL, subgroup.ID)
		resp := h.MakeRequest("DELETE", url, nil)
		resp.Body.Close()
	}
}
