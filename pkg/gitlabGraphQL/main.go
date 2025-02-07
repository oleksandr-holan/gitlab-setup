package gitlabGraphQL

import (
	"context"
	"fmt"

	"github.com/hasura/go-graphql-client"
)

// GraphQL types for the mutation
type ProjectTargetBranchRuleCreateInput struct {
	ProjectID    string `json:"projectId"`
	Name         string `json:"name"`
	TargetBranch string `json:"targetBranch"`
}

// ProjectTargetBranchRule represents a single target branch rule
type ProjectTargetBranchRule struct {
	ID           string
	Name         string
	TargetBranch string
	CreatedAt    string
}

func CreateTargetBranchRule(ctx context.Context, client *graphql.Client, projectID int, name, targetBranch string) error {
	var mutation struct {
		ProjectTargetBranchRuleCreate struct {
			Errors           []string
			TargetBranchRule struct {
				Name string
			}
		} `graphql:"projectTargetBranchRuleCreate(input: $input)"`
	}

	variables := map[string]interface{}{
		"input": ProjectTargetBranchRuleCreateInput{
			ProjectID:    fmt.Sprintf("gid://gitlab/Project/%d", projectID),
			Name:         name,
			TargetBranch: targetBranch,
		},
	}

	err := client.Mutate(ctx, &mutation, variables)
	if err != nil {
		return fmt.Errorf("GraphQL mutation failed: %v", err)
	}

	if len(mutation.ProjectTargetBranchRuleCreate.Errors) > 0 {
		return fmt.Errorf("GraphQL errors: %v", mutation.ProjectTargetBranchRuleCreate.Errors)
	}

	return nil
}

func GetTargetBranchRules(ctx context.Context, client *graphql.Client, projectPath string) ([]ProjectTargetBranchRule, error) {
	var query struct {
		Project struct {
			TargetBranchRules struct {
				Nodes []ProjectTargetBranchRule
			} `graphql:"targetBranchRules"`
		} `graphql:"project(fullPath: $fullPath)"`
	}

	variables := map[string]interface{}{
		"fullPath": graphql.ID(projectPath),
	}

	err := client.Query(ctx, &query, variables)
	if err != nil {
		return nil, fmt.Errorf("GraphQL query failed: %v", err)
	}

	return query.Project.TargetBranchRules.Nodes, nil
}
