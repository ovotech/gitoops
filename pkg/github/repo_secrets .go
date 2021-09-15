package github

import (
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/ovotech/gitoops/pkg/database"
)

type RepoSecretsIngestor struct {
	restclient *RESTClient
	db         *database.Database
	data       *RepoSecretsData
	repoName   string
	repoId     int64
	session    string
}

type RepoSecretsData struct {
	Secrets []struct {
		Name string `json:"name"`
	} `json:"secrets"`
}

func (ing *RepoSecretsIngestor) Sync() {
	ing.fetchData()
	ing.insertRepoSecrets()
}

func (ing *RepoSecretsIngestor) fetchData() {
	query := fmt.Sprintf("repos/%s/%s/actions/secrets", ing.restclient.organization, ing.repoName)

	data := ing.restclient.fetch(query)
	json.Unmarshal(data, &ing.data)
}

func (ing *RepoSecretsIngestor) insertRepoSecrets() {
	envVars := []map[string]interface{}{}

	for _, envVar := range ing.data.Secrets {
		strRepoId := fmt.Sprintf("%d", ing.repoId)
		id := fmt.Sprintf("%x", md5.Sum([]byte(strRepoId+envVar.Name)))
		envVars = append(envVars, map[string]interface{}{
			"id":     id,
			"name":   envVar.Name,
			"repoId": ing.repoId,
		})
	}

	ing.db.Run(`
	UNWIND $envVars AS envVar

	MERGE (v:EnvironmentVariable{id: envVar.id})

	SET v.name = envVar.name,
	v.session = $session

	WITH v, envVar

	MATCH (r:Repository{databaseId: envVar.repoId})
	MERGE (r)-[rel:EXPOSES_ENVIRONMENT_VARIABLE]->(v)
	SET rel.session = $session
	`, map[string]interface{}{"envVars": envVars, "session": ing.session})
}
