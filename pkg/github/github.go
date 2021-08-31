package github

import (
	"net/http"

	"github.com/ovotech/gitoops/pkg/database"
	log "github.com/sirupsen/logrus"
)

type GitHub struct {
	gqlclient  *GraphQLClient
	restclient *RESTClient
	db         *database.Database
}

func GetGitHub(db *database.Database, token, organization string) *GitHub {
	return &GitHub{
		gqlclient: &GraphQLClient{
			client:       &http.Client{},
			token:        token,
			organization: organization,
		},
		restclient: &RESTClient{
			client:       &http.Client{},
			token:        token,
			organization: organization,
		},
		db: db,
	}
}

// Takes a slice of ingestor names and calls those in the right order.
func (g *GitHub) SyncByIngestorNames(targetIngestors []string) {
	log.Infof("Syncing with these ingestors: %s", targetIngestors)

	// orgIngestors query at the org level, as opposed to querying a specific team or repo.
	// nb: order matters for these!
	orgIngestorOrderedKeys := []string{"organizations", "teams", "users", "repos"}
	orgIngestors := map[string]Ingestor{
		"organizations": &OrganizationsIngestor{gqlclient: g.gqlclient, db: g.db},
		"teams":         &TeamsIngestor{gqlclient: g.gqlclient, db: g.db},
		"users":         &UsersIngestor{gqlclient: g.gqlclient, db: g.db},
		"repos":         &ReposIngestor{gqlclient: g.gqlclient, db: g.db},
	}

	for _, name := range orgIngestorOrderedKeys {
		if !sliceContains(targetIngestors, name) {
			continue
		}
		log.Infof("Running org ingestor %s", name)
		orgIngestors[name].FetchData()
		orgIngestors[name].Sync()
	}

	// teamIngestors query at a specific team level
	teamRecords := g.db.Run(`MATCH (t:Team) RETURN t.slug as teamSlug`, map[string]interface{}{})
	for teamRecords.Next() {
		teamSlug, _ := teamRecords.Record().Get("teamSlug")

		teamIngestors := map[string]Ingestor{
			"teamrepos": &TeamReposIngestor{
				gqlclient: g.gqlclient,
				db:        g.db,
				teamSlug:  teamSlug.(string),
			},
			"teammembers": &TeamMembersIngestor{
				gqlclient: g.gqlclient,
				db:        g.db,
				teamSlug:  teamSlug.(string),
			},
		}

		for name, ingestor := range teamIngestors {
			if !sliceContains(targetIngestors, name) {
				continue
			}
			log.Infof("Running team ingestor %s on team %s", name, teamSlug)
			ingestor.FetchData()
			ingestor.Sync()
		}
	}

	// repoIngestors query at a specific repo level
	repoRecords := g.db.Run(
		`MATCH (r:Repository) RETURN r.name as repoName`,
		map[string]interface{}{},
	)
	for repoRecords.Next() {
		repoName, _ := repoRecords.Record().Get("repoName")

		repoIngestors := map[string]Ingestor{
			"repowebhooks": &RepoWebhooksIngestor{
				restclient: g.restclient,
				db:         g.db,
				repoName:   repoName.(string),
			},
		}

		for name, ingestor := range repoIngestors {
			if !sliceContains(targetIngestors, name) {
				continue
			}
			log.Infof("Running repo ingestor %s on repo %s", name, repoName)
			ingestor.FetchData()
			ingestor.Sync()
		}
	}
}

// Returns true if slice s contains element e, false otherwise.
func sliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
