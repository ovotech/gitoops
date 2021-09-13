package github

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/ovotech/gitoops/pkg/database"
)

type OrganizationSecretsIngestor struct {
	restclient *RESTClient
	db         *database.Database
	data       *OrganizationSecretsData
	session    string
}

type OrganizationSecretsData struct {
	Secrets []struct {
		Name                    string    `json:"name"`
		CreatedAt               time.Time `json:"created_at"`
		Visibility              string    `json:"visibility"`
		SelectedRepositoriesURL string    `json:"selected_repositories_url,omitempty"`
	} `json:"secrets"`
}

type OrganizationSecretsSelectedRepositories struct {
	Repositories []struct {
		HTMLURL string `json:"html_url"`
	} `json:"repositories"`
}

func (ing *OrganizationSecretsIngestor) Sync() {
	ing.fetchData()
	ing.insertAllRepositoriesSecrets()
	ing.insertPrivateRepositoriesSecrets()
	ing.insertSelectedRepositoriesSecrets()
}

func (ing *OrganizationSecretsIngestor) fetchData() {
	query := fmt.Sprintf("orgs/%s/actions/secrets", ing.restclient.organization)

	data := ing.restclient.fetch(query)
	json.Unmarshal(data, &ing.data)
}

func (ing *OrganizationSecretsIngestor) insertAllRepositoriesSecrets() {
	secrets := []map[string]interface{}{}

	for _, secret := range ing.data.Secrets {
		if secret.Visibility != "all" {
			continue
		}
		id := fmt.Sprintf("%x", md5.Sum([]byte(secret.CreatedAt.String()+secret.Name)))
		secrets = append(secrets, map[string]interface{}{
			"id":   id,
			"name": secret.Name,
		})
	}

	ing.db.Run(`
	UNWIND $secrets AS secret

	MERGE (v:EnvironmentVariable{id: secret.id})

	SET v.name = secret.name,
	v.session = $session

	WITH v

	MATCH (r:Repository)
	MERGE (r)-[rel:EXPOSES_ENVIRONMENT_VARIABLE]->(v)
	SET rel.session = $session
	`, map[string]interface{}{"secrets": secrets, "session": ing.session})
}

func (ing *OrganizationSecretsIngestor) insertPrivateRepositoriesSecrets() {
	secrets := []map[string]interface{}{}

	for _, secret := range ing.data.Secrets {
		if secret.Visibility != "private" {
			continue
		}
		id := fmt.Sprintf("%x", md5.Sum([]byte(secret.CreatedAt.String()+secret.Name)))
		secrets = append(secrets, map[string]interface{}{
			"id":   id,
			"name": secret.Name,
		})
	}

	ing.db.Run(`
	UNWIND $secrets AS secret

	MERGE (v:EnvironmentVariable{id: secret.id})

	SET v.name = secret.name,
	v.session = $session

	WITH v

	MATCH (r:Repository{isPrivate:TRUE})
	MERGE (r)-[rel:EXPOSES_ENVIRONMENT_VARIABLE]->(v)
	SET rel.session = $session
	`, map[string]interface{}{"secrets": secrets, "session": ing.session})
}

func (ing *OrganizationSecretsIngestor) insertSelectedRepositoriesSecrets() {
	secrets := []map[string]interface{}{}

	for _, secret := range ing.data.Secrets {
		if secret.Visibility != "selected" {
			continue
		}

		// fetch list of repositories
		u, _ := url.Parse(secret.SelectedRepositoriesURL)
		data := ing.restclient.fetch(u.Path)
		selectedRepositories := OrganizationSecretsSelectedRepositories{}
		json.Unmarshal(data, &selectedRepositories)

		id := fmt.Sprintf("%x", md5.Sum([]byte(secret.CreatedAt.String()+secret.Name)))
		for _, repository := range selectedRepositories.Repositories {
			secrets = append(secrets, map[string]interface{}{
				"id":      id,
				"name":    secret.Name,
				"repoURL": repository.HTMLURL,
			})
		}
	}

	ing.db.Run(`
	UNWIND $secrets AS secret

	MERGE (v:EnvironmentVariable{id: secret.id})

	SET v.name = secret.name,
	v.session = $session

	WITH v, secret

	MATCH (r:Repository{id: secret.repoURL})
	MERGE (r)-[rel:EXPOSES_ENVIRONMENT_VARIABLE]->(v)
	SET rel.session = $session
	`, map[string]interface{}{"secrets": secrets, "session": ing.session})
}
