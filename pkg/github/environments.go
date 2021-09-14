package github

import (
	"encoding/json"
	"fmt"

	"github.com/ovotech/gitoops/pkg/database"
)

type EnvironmentsIngestor struct {
	restclient *RESTClient
	db         *database.Database
	data       *EnvironmentsData
	repoName   string
	session    string
}

type EnvironmentsData struct {
	Environments []struct {
		Name            string `json:"name"`
		HTMLURL         string `json:"html_url"`
		ProtectionRules []struct {
			ID     int    `json:"id"`
			NodeID string `json:"node_id"`
			Type   string `json:"type"`
		} `json:"protection_rules"`
		DeploymentBranchPolicy struct {
			ProtectedBranches    bool `json:"protected_branches"`
			CustomBranchPolicies bool `json:"custom_branch_policies"`
		} `json:"deployment_branch_policy"`
	} `json:"environments"`
}

func (ing *EnvironmentsIngestor) Sync() {
	ing.fetchData()
	ing.insertEnvironments()
}

func (ing *EnvironmentsIngestor) fetchData() {
	query := fmt.Sprintf("repos/%s/%s/environments", ing.restclient.organization, ing.repoName)

	data := ing.restclient.fetch(query)
	json.Unmarshal(data, &ing.data)
}

func (ing *EnvironmentsIngestor) insertEnvironments() {
	environments := []map[string]interface{}{}

	for _, environment := range ing.data.Environments {
		environments = append(environments, map[string]interface{}{
			"url":                environment.HTMLURL,
			"name":               environment.Name,
			"repoName":           ing.repoName,
			"protectedBranches":  environment.DeploymentBranchPolicy.ProtectedBranches,
			"customBranchPolicy": environment.DeploymentBranchPolicy.CustomBranchPolicies,
		})
	}

	ing.db.Run(`
	UNWIND $environments AS environment

	MERGE (e:Environment{id: environment.url})

	SET e.name = environment.name,
	e.url = environment.url,
	e.protectedBranches = environment.protectedBranches,
	e.customBranchPolicy = environment.customBranchPolicy,
	e.session = $session

	WITH e, environment

	MATCH (r:Repository{name: environment.repoName})
	MERGE (r)-[rel:HAS_ENVIRONMENT]->(e)
	SET rel.session = $session
	`, map[string]interface{}{"environments": environments, "session": ing.session})
}
