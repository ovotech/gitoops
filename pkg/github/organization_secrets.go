package github

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
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

func (ing *OrganizationSecretsIngestor) Sync() {
	ing.fetchData()
	ing.insertAllRepositoriesSecrets()
}

func (ing *OrganizationSecretsIngestor) fetchData() {
	query := fmt.Sprintf("orgs/%s/actions/secrets", ing.restclient.organization)

	data := ing.restclient.fetch(query)
	json.Unmarshal(data, &ing.data)
	fmt.Println(ing.data)
}

func (ing *OrganizationSecretsIngestor) insertAllRepositoriesSecrets() {
	secrets := []map[string]interface{}{}

	for _, secret := range ing.data.Secrets {
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
