package testutil

import (
	"fmt"
	"math/rand"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (h *GitLabTestHelper) CreateTestStructure() ([]*gitlab.Group, []*gitlab.Project) {
	// Create source subgroups
	subGroup1 := h.CreateTestGroup(h.Config.MainGroupID, h.GenerateRandomString())
	subGroup2 := h.CreateTestGroup(subGroup1.ID, h.GenerateRandomString())

	// Create target subgroups for forks (in the same main group)
	forkSubGroup1 := h.CreateTestGroup(h.Config.MainGroupID, fmt.Sprintf("fork-%s", subGroup1.Path))
	forkSubGroup2 := h.CreateTestGroup(forkSubGroup1.ID, fmt.Sprintf("fork-%s", subGroup2.Path))
	subgroups := []*gitlab.Group{subGroup1, subGroup2, forkSubGroup1, forkSubGroup2}

	projectConfigs := []struct {
		groupID      int
		squashOption gitlab.SquashOptionValue
		forkToGroup  int
	}{
		{subGroup2.ID, gitlab.SquashOptionNever, forkSubGroup2.ID},
		{subGroup1.ID, gitlab.SquashOptionAlways, forkSubGroup1.ID},
		{h.Config.MainGroupID, gitlab.SquashOptionDefaultOff, 0},
	}

	projects := make([]*gitlab.Project, 0, len(projectConfigs))
	for _, config := range projectConfigs {
		project := h.CreateTestProject(config.groupID, h.GenerateRandomString(), config.squashOption)

		_, _, err := h.Client.Branches.CreateBranch(project.ID, &gitlab.CreateBranchOptions{
			Branch: gitlab.Ptr("environment/dev"),
			Ref:    gitlab.Ptr("main"),
		})
		if err != nil {
			h.T.Errorf("Failed to create branch: %v", err)
		}

		projects = append(projects, project)

		if config.forkToGroup == 0 {
			continue
		}

		forkOpts := &gitlab.ForkProjectOptions{
			NamespaceID: gitlab.Ptr(config.forkToGroup),
			Name:        gitlab.Ptr(fmt.Sprintf("fork-%s", project.Path)),
		}

		forkedProject, _, err := h.Client.Projects.ForkProject(project.ID, forkOpts)
		if err != nil {
			h.T.Fatalf("Failed to fork project:: %v", err)
		}

		projects = append(projects, forkedProject)
	}

	return subgroups, projects
}

func (h *GitLabTestHelper) CreateTestGroup(parentID int, name string) *gitlab.Group {
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

func (h *GitLabTestHelper) CreateTestProject(groupID int, name string, squashOption gitlab.SquashOptionValue) *gitlab.Project {
	h.T.Helper()

	createOpts := &gitlab.CreateProjectOptions{
		Name:                      gitlab.Ptr(name),
		Path:                      gitlab.Ptr(name),
		NamespaceID:               gitlab.Ptr(groupID),
		SquashOption:              gitlab.Ptr(squashOption),
		AutocloseReferencedIssues: gitlab.Ptr(false),
		MergeMethod:               gitlab.Ptr(gitlab.NoFastForwardMerge),
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
			h.T.Errorf("Failed to delete project %d: %v", project.ID, err)
		}
	}

	for _, subgroup := range subgroups {
		_, err := h.Client.Groups.DeleteGroup(subgroup.ID, nil)
		if err != nil {
			h.T.Errorf("Failed to delete subgroup %d: %v", subgroup.ID, err)
		}
	}
}
