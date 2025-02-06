package testutil

import (
	"math/rand"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (h *GitLabTestHelper) CreateTestStructure() ([]*gitlab.Group, []*gitlab.Project) {
	subGroup1 := h.CreateGroup(h.Config.MainGroupID, h.GenerateRandomString())
	subGroup2 := h.CreateGroup(subGroup1.ID, h.GenerateRandomString())
	subgroups := []*gitlab.Group{subGroup1, subGroup2}

	projects := []*gitlab.Project{
		h.CreateProject(subGroup2.ID, h.GenerateRandomString(), "never"),
		h.CreateProject(subGroup1.ID, h.GenerateRandomString(), "always"),
		h.CreateProject(h.Config.MainGroupID, h.GenerateRandomString(), "default_off"),
	}

	return subgroups, projects
}

func (h *GitLabTestHelper) CreateGroup(parentID int, name string) *gitlab.Group {
	h.T.Helper()

	createOpts := &gitlab.CreateGroupOptions{
		Name: gitlab.Ptr(name),
		Path: gitlab.Ptr(name),
	}
	if parentID != 0 {
		createOpts.ParentID = gitlab.Ptr(parentID)
	}

	group, _, err := h.Client.Groups.CreateGroup(createOpts)
	if err != nil {
		h.T.Fatalf("Failed to create group: %v", err)
	}

	return group
}

func (h *GitLabTestHelper) CreateProject(groupID int, name, squashOption string) *gitlab.Project {
	h.T.Helper()

	createOpts := &gitlab.CreateProjectOptions{
		Name:         gitlab.Ptr(name),
		Path:         gitlab.Ptr(name),
		NamespaceID:  gitlab.Ptr(groupID),
		SquashOption: gitlab.Ptr(gitlab.SquashOptionValue(squashOption)),
	}

	project, _, err := h.Client.Projects.CreateProject(createOpts)
	if err != nil {
		h.T.Fatalf("Failed to create project: %v", err)
	}

	return project
}

func (h *GitLabTestHelper) CleanupTestStructure(subgroups []*gitlab.Group, projects []*gitlab.Project) {
	h.T.Helper()
	h.CleanupSubgroupsAndProjects(subgroups, projects)
}

func (h *GitLabTestHelper) GenerateRandomString() string {
	b := make([]byte, stringLength)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func (h *GitLabTestHelper) CleanupSubgroupsAndProjects(subgroups []*gitlab.Group, projects []*gitlab.Project) {
	h.T.Helper()

	for _, project := range projects {
		_, err := h.Client.Projects.DeleteProject(project.ID, nil)
		if err != nil {
			h.T.Fatalf("Failed to delete project %d: %v", project.ID, err)
		}
	}

	for _, subgroup := range subgroups {
		_, err := h.Client.Groups.DeleteGroup(subgroup.ID, nil)
		if err != nil {
			h.T.Fatalf("Failed to delete subgroup %d: %v", subgroup.ID, err)
		}
	}
}
