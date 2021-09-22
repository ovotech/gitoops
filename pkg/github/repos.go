package github

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ovotech/gitoops/pkg/database"
)

type ReposIngestor struct {
	gqlclient *GraphQLClient
	db        *database.Database
	data      *ReposData
	session   string
}

type ReposData struct {
	Nodes []struct {
		DatabaseId    int    `json:"databaseId"`
		URL           string `json:"url"`
		Name          string `json:"name"`
		IsPrivate     bool   `json:"isPrivate"`
		IsArchived    bool   `json:"isArchived"`
		Collaborators struct {
			Edges []struct {
				Permission string `json:"permission"`
			} `json:"edges"`
			Nodes []struct {
				URL   string `json:"url"`
				Login string `json:"login"`
			} `json:"nodes"`
		} `json:"collaborators"`
		CircleCI struct {
			Text string `json:"text"`
		} `json:"circleci"`
		Travis struct {
			Text string `json:"text"`
		} `json:"travis"`
		Jenkins struct {
			Text string `json:"text"`
		} `json:"jenkins"`
		CodeBuild struct {
			Text string `json:"text"`
		} `json:"codebuild"`
		CloudBuild struct {
			Text string `json:"text"`
		} `json:"cloudbuild"`
		BuildSBT struct {
			Text string `json:"text"`
		} `json:"buildsbt"`
		Codeowners struct {
			Text string `json:"text"`
		} `json:"codeowners"`
		Actions struct {
			Entries []struct {
				Name   string `json:"name"`
				Object struct {
					Text string `json:"text"`
				} `json:"object"`
			} `json:"entries"`
		} `json:"actions"`
		BranchProtectionRules struct {
			Nodes []struct {
				Pattern                  string `json:"pattern"`
				RequiresApprovingReviews bool   `json:"requiresApprovingReviews"`
			} `json:"nodes"`
		} `json:"branchProtectionRules"`
		PullRequests struct {
			Nodes []struct {
				Commits struct {
					Nodes []struct {
						Commit struct {
							Status struct {
								Contexts []struct {
									TargetURL   string `json:"targetUrl"`
									Description string `json:"description"`
									Context     string `json:"context"`
								} `json:"contexts"`
							} `json:"status"`
						} `json:"commit"`
					} `json:"nodes"`
				} `json:"commits"`
			} `json:"nodes"`
		} `json:"pullRequests"`
		DefaultBranchRef struct {
			Target struct {
				History struct {
					Edges []struct {
						Node struct {
							Status struct {
								Contexts []struct {
									Context     string `json:"context"`
									TargetURL   string `json:"targetUrl"`
									Description string `json:"description"`
								} `json:"contexts"`
							} `json:"status"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"history"`
			} `json:"target"`
		} `json:"defaultBranchRef"`
	} `json:"nodes"`
}

func (ing *ReposIngestor) Sync() {
	ing.fetchData()
	ing.insertRepos()
	ing.insertReposFiles()
	ing.insertReposCollaborators()
	ing.insertReposPullRequestsStatusChecks()
	ing.insertReposDefaultBranchStatusChecks()
	ing.insertReposBranchProtectionRules()
}

func (ing *ReposIngestor) fetchData() {
	query := `
	query($login: String!, $cursor: String) {
		organization(login: $login) {
			repositories(first: 10, after: $cursor) {
				pageInfo {
					endCursor
					hasNextPage
				}
				nodes {
					databaseId
					url
					name
					isPrivate
					isArchived
					collaborators(affiliation: DIRECT, first: 100) {
						edges {
							permission
						}
						nodes {
							url
							login
						}
					}
					circleci: object(expression: "HEAD:.circleci/config.yml") {
						... on Blob {
							text
						}
					}
					travis: object(expression: "HEAD:.travis.yml") {
						... on Blob {
							text
						}
					}
					jenkins: object(expression: "HEAD:Jenkinsfile") {
						... on Blob {
							text
						}
					}
					codebuild: object(expression: "HEAD:buildspec.yml") {
						... on Blob {
							text
						}
					}
					cloudbuild: object(expression: "HEAD:cloudbuild.yaml") {
						... on Blob {
							text
						}
					}
					buildsbt: object(expression: "HEAD:build.sbt") {
						... on Blob {
							text
						}
					}
					codeowners:object(expression: "HEAD:.github/CODEOWNERS") {
						... on Blob {
							text
						}
					}
					actions:object(expression: "HEAD:.github/workflows") {
						... on Tree {
							entries {
								name
								object {
									... on Blob {
										text
									}
								}
							}
						}
					}
					branchProtectionRules(first: 5) {
						nodes {
							pattern
							requiresApprovingReviews
						}
          			}
					pullRequests(last: 10) {
						nodes {
							commits(last: 1) {
								nodes {
									commit {
										status {
											contexts {
												targetUrl
												description
												context
											}
										}
									}
								}
							}
						}
					}
					defaultBranchRef {
						target {
							... on Commit {
								history(first: 10) {
									edges {
										node {
											status {
												contexts {
													context
													targetUrl
													description
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	`

	data := ing.gqlclient.fetch(
		query,
		"organization.repositories",
		map[string]string{},
	)

	json.Unmarshal(data, &ing.data)
}

func (ing *ReposIngestor) insertRepos() {
	repos := []map[string]interface{}{}

	for _, repoNode := range ing.data.Nodes {
		repos = append(repos, map[string]interface{}{
			"databaseId":   repoNode.DatabaseId,
			"url":          repoNode.URL,
			"name":         repoNode.Name,
			"isPrivate":    repoNode.IsPrivate,
			"isArchived":   repoNode.IsArchived,
			"organization": ing.gqlclient.organization,
		})
	}

	ing.db.Run(`
	UNWIND $repos as repo

	MERGE (r:Repository{id: repo.url})

	SET r.url = repo.url,
	r.databaseId = repo.databaseId,
	r.name = repo.name,
	r.isPrivate = repo.isPrivate,
	r.isArchived = repo.isArchived,
	r.session = $session

	WITH r, repo

	MATCH (o:Organization{login: repo.organization})
	MERGE (r)-[rel:OWNED_BY]->(o)
	SET rel.session = $session
	`, map[string]interface{}{"repos": repos, "session": ing.session})
}

func (ing *ReposIngestor) insertReposCollaborators() {
	reposCollaborators := []map[string]interface{}{}

	for _, repoNode := range ing.data.Nodes {
		for j, collaboratorNode := range repoNode.Collaborators.Nodes {
			collaboratorEdge := repoNode.Collaborators.Edges[j]
			reposCollaborators = append(reposCollaborators, map[string]interface{}{
				"url":        collaboratorNode.URL,
				"login":      collaboratorNode.Login,
				"permission": collaboratorEdge.Permission,
				"repoID":     repoNode.URL,
			})
		}
	}

	ing.db.Run(`
	UNWIND $reposCollaborators as repoCollaborator

	MERGE (u:User{id: repoCollaborator.url})

	SET u.login = repoCollaborator.login,
	u.session = $session

	WITH u, repoCollaborator

	MATCH (r:Repository{id: repoCollaborator.repoID})
	MERGE (u)-[rel:HAS_PERMISSION_ON{permission: repoCollaborator.permission}]->(r)
	SET rel.session = $session
	`, map[string]interface{}{"reposCollaborators": reposCollaborators, "session": ing.session})
}

func (ing *ReposIngestor) insertReposFiles() {
	reposFiles := []map[string]interface{}{}

	for _, repoNode := range ing.data.Nodes {
		if repoNode.CircleCI.Text != "" {
			id := fmt.Sprintf("%x", md5.Sum([]byte("circleci"+repoNode.URL)))
			reposFiles = append(reposFiles, map[string]interface{}{
				"id":     id,
				"path":   ".circleci/config.yml",
				"text":   repoNode.CircleCI.Text,
				"repoID": repoNode.URL,
			})
		}
		if repoNode.Travis.Text != "" {
			id := fmt.Sprintf("%x", md5.Sum([]byte("travis"+repoNode.URL)))
			reposFiles = append(reposFiles, map[string]interface{}{
				"id":     id,
				"path":   ".travis.yml",
				"text":   repoNode.Travis.Text,
				"repoID": repoNode.URL,
			})
		}
		if repoNode.Jenkins.Text != "" {
			id := fmt.Sprintf("%x", md5.Sum([]byte("jenkins"+repoNode.URL)))
			reposFiles = append(reposFiles, map[string]interface{}{
				"id":     id,
				"path":   "Jenkinsfile",
				"text":   repoNode.Jenkins.Text,
				"repoID": repoNode.URL,
			})
		}
		if repoNode.CodeBuild.Text != "" {
			id := fmt.Sprintf("%x", md5.Sum([]byte("codebuild"+repoNode.URL)))
			reposFiles = append(reposFiles, map[string]interface{}{
				"id":     id,
				"path":   "buildspec.yml",
				"text":   repoNode.CodeBuild.Text,
				"repoID": repoNode.URL,
			})
		}
		if repoNode.CloudBuild.Text != "" {
			id := fmt.Sprintf("%x", md5.Sum([]byte("cloudbuild"+repoNode.URL)))
			reposFiles = append(reposFiles, map[string]interface{}{
				"id":     id,
				"path":   "cloudbuild.yaml",
				"text":   repoNode.CloudBuild.Text,
				"repoID": repoNode.URL,
			})
		}
		if repoNode.BuildSBT.Text != "" {
			id := fmt.Sprintf("%x", md5.Sum([]byte("buildsbt"+repoNode.URL)))
			reposFiles = append(reposFiles, map[string]interface{}{
				"id":     id,
				"path":   "build.sbt",
				"text":   repoNode.BuildSBT.Text,
				"repoID": repoNode.URL,
			})
		}
		if repoNode.Codeowners.Text != "" {
			id := fmt.Sprintf("%x", md5.Sum([]byte("codeowners"+repoNode.URL)))
			reposFiles = append(reposFiles, map[string]interface{}{
				"id":     id,
				"path":   ".github/CODEOWNERS",
				"text":   repoNode.Codeowners.Text,
				"repoID": repoNode.URL,
			})
		}
		if len(repoNode.Actions.Entries) > 0 {
			for _, entry := range repoNode.Actions.Entries {
				id := fmt.Sprintf("%x", md5.Sum([]byte("actions"+entry.Name+repoNode.URL)))
				reposFiles = append(reposFiles, map[string]interface{}{
					"id":     id,
					"path":   ".github/workflows/" + entry.Name,
					"text":   entry.Object.Text,
					"repoID": repoNode.URL,
				})
			}
		}
	}

	ing.db.Run(`
	UNWIND $reposFiles as repoFile

	MERGE (f:File{id: repoFile.id})

	SET f.path = repoFile.path,
	f.text = repoFile.text,
	f.session = $session

	WITH f, repoFile

	MATCH (r:Repository{id: repoFile.repoID})
	MERGE (r)-[rel:HAS_CI_CONFIGURATION_FILE]->(f)
	SET rel.session = $session
	`, map[string]interface{}{"reposFiles": reposFiles, "session": ing.session})
}

func (ing *ReposIngestor) insertReposPullRequestsStatusChecks() {
	// We don't need to map the full hierarchy with pull requests, commits etc. since we're only
	// interested CI integrations. We therefor have a notion of StatusCheck node that holds
	// information such as the context (i.e. "ci/circleci: Build Error") and the hostname of the
	// integration that ran the status check.
	// This node is unique per repo, so even if two repos have the same context/host combination for
	// a status check, they won't point to the same node.
	reposStatusChecks := []map[string]interface{}{}

	for _, repoNode := range ing.data.Nodes {
		for _, pullRequestNode := range repoNode.PullRequests.Nodes {
			if len(pullRequestNode.Commits.Nodes) == 0 {
				continue
			}
			for _, context := range pullRequestNode.Commits.Nodes[0].Commit.Status.Contexts {
				u, _ := url.Parse(context.TargetURL)
				id := fmt.Sprintf("%x", md5.Sum([]byte(context.Context+repoNode.URL)))
				reposStatusChecks = append(reposStatusChecks, map[string]interface{}{
					// We use a md5 sum of context and repo url for the status check id
					// This ensures we have a unique status check node per repo
					"id":      id,
					"repoID":  repoNode.URL,
					"context": context.Context,
					"host":    u.Host,
				})
			}
		}
	}

	ing.db.Run(`
	UNWIND $reposStatusChecks as repoStatusCheck

	MERGE (s:StatusCheck{id: repoStatusCheck.id})

	SET s.context = repoStatusCheck.context,
	s.host = repoStatusCheck.host,
	s.session = $session

	WITH s, repoStatusCheck

	MATCH (r:Repository{id: repoStatusCheck.repoID})
	MERGE (r)-[rel:HAS_STATUS_CHECK{pullRequest: true}]->(s)
	SET rel.session = $session
	`, map[string]interface{}{"reposStatusChecks": reposStatusChecks, "session": ing.session})
}

func (ing *ReposIngestor) insertReposDefaultBranchStatusChecks() {
	reposStatusChecks := []map[string]interface{}{}

	for _, repoNode := range ing.data.Nodes {
		for _, statusEdge := range repoNode.DefaultBranchRef.Target.History.Edges {
			for _, context := range statusEdge.Node.Status.Contexts {
				u, _ := url.Parse(context.TargetURL)
				id := fmt.Sprintf("%x", md5.Sum([]byte(context.Context+repoNode.URL)))
				reposStatusChecks = append(reposStatusChecks, map[string]interface{}{
					// We use a md5 sum of context and repo url for the status check id
					// This ensures we have a unique status check node per repo
					"id":      id,
					"repoID":  repoNode.URL,
					"context": context.Context,
					"host":    u.Host,
				})
			}
		}
	}

	ing.db.Run(`
	UNWIND $reposStatusChecks as repoStatusCheck

	MERGE (s:StatusCheck{id: repoStatusCheck.id})

	SET s.context = repoStatusCheck.context,
	s.host = repoStatusCheck.host,
	s.session = $session

	WITH s, repoStatusCheck

	MATCH (r:Repository{id: repoStatusCheck.repoID})
	MERGE (r)-[rel:HAS_STATUS_CHECK{defaultBranch: true}]->(s)
	SET rel.session = $session
	`, map[string]interface{}{"reposStatusChecks": reposStatusChecks, "session": ing.session})
}

func (ing *ReposIngestor) insertReposBranchProtectionRules() {
	reposBranchProtectionRules := []map[string]interface{}{}

	for _, repoNode := range ing.data.Nodes {
		for _, ruleNode := range repoNode.BranchProtectionRules.Nodes {
			// nb: branch protection patterns are unique per repo
			id := fmt.Sprintf("%x", md5.Sum([]byte(ruleNode.Pattern+repoNode.URL)))
			reposBranchProtectionRules = append(
				reposBranchProtectionRules,
				map[string]interface{}{
					"id":              id,
					"repoID":          repoNode.URL,
					"pattern":         ruleNode.Pattern,
					"requiresReviews": ruleNode.RequiresApprovingReviews,
				},
			)
		}
	}

	ing.db.Run(`
	UNWIND $reposBranchProtectionRules as rule

	MERGE (b:BranchProtectionRule{id: rule.id})

	SET b.pattern = rule.pattern,
	b.requiresReviews = rule.requiresReviews,
	b.session = $session

	WITH b, rule
	MATCH (r:Repository{id: rule.repoID})
	MERGE (r)-[rel:HAS_BRANCH_PROTECTION_RULE]->(b)
	SET rel.session = $session
	`, map[string]interface{}{"reposBranchProtectionRules": reposBranchProtectionRules, "session": ing.session})
}
