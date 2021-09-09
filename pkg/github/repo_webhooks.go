package github

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/ovotech/gitoops/pkg/database"
)

type RepoWebhooksIngestor struct {
	restclient *RESTClient
	db         *database.Database
	data       *RepoWebhooksData
	repoName   string
	session    string
}

type RepoWebhooksData []struct {
	Active bool `json:"active"`
	Config struct {
		ContentType string `json:"content_type"`
		InsecureSsl string `json:"insecure_ssl"`
		Secret      string `json:"secret"`
		URL         string `json:"url"`
	} `json:"config"`
	CreatedAt     time.Time `json:"created_at"`
	DeliveriesURL string    `json:"deliveries_url"`
	Events        []string  `json:"events"`
	ID            int       `json:"id"`
	LastResponse  struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"last_response"`
	Name      string    `json:"name"`
	PingURL   string    `json:"ping_url"`
	TestURL   string    `json:"test_url"`
	Type      string    `json:"type"`
	UpdatedAt time.Time `json:"updated_at"`
	URL       string    `json:"url"`
}

func (ing *RepoWebhooksIngestor) Sync() {
	ing.fetchData()
	ing.insertRepoWebhooks()
}

func (ing *RepoWebhooksIngestor) fetchData() {
	query := fmt.Sprintf("repos/%s/%s/hooks", ing.restclient.organization, ing.repoName)

	data := ing.restclient.fetch(query)
	json.Unmarshal(data, &ing.data)
}

func (ing *RepoWebhooksIngestor) insertRepoWebhooks() {
	webhooks := []map[string]interface{}{}

	for _, webhook := range *ing.data {
		u, _ := url.Parse(webhook.Config.URL)
		webhooks = append(webhooks, map[string]interface{}{
			"url":      webhook.URL,
			"target":   webhook.Config.URL,
			"name":     webhook.Name,
			"events":   webhook.Events,
			"host":     u.Host,
			"repoName": ing.repoName,
		})
	}

	ing.db.Run(`
	UNWIND $webhooks AS webhook

	MERGE (w:Webhook{id: webhook.url})

	SET w.name = webhook.name,
	w.url = webhook.url,
	w.target = webhook.target,
	w.host = webhook.host,
	w.events = webhook.events

	WITH w, webhook

	MATCH (r:Repository{name: webhook.repoName})
	MERGE (r)-[rel:HAS_WEBHOOK]->(w)
	`, map[string]interface{}{"webhooks": webhooks})
}
