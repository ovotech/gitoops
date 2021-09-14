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
	session    string
}

func GetGitHub(db *database.Database, token, organization, session string) *GitHub {
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
		db:      db,
		session: session,
	}
}

// Sync with default ingestors.
func (g *GitHub) Sync() {
	ingestors := []string{"organizations", "teams", "users", "repos", "teamrepos", "teammembers"}
	g.SyncByIngestorNames(ingestors)
}

// Takes a slice of ingestor names and calls those in the right order.
func (g *GitHub) SyncByIngestorNames(targetIngestors []string) {
	log.Infof("Syncing with these ingestors: %s", targetIngestors)

	// orgIngestors query at the org level, as opposed to querying a specific team or repo.
	// nb: order matters for these!
	orgIngestorOrderedKeys := []string{
		"organizations",
		"teams",
		"users",
		"repos",
		"organizationsecrets",
	}
	orgIngestors := map[string]Ingestor{
		"organizations": &OrganizationsIngestor{
			gqlclient: g.gqlclient,
			db:        g.db,
			data:      &OrganizationsData{},
			session:   g.session,
		},
		"teams": &TeamsIngestor{
			gqlclient: g.gqlclient,
			db:        g.db,
			data:      &TeamsData{},
			session:   g.session,
		},
		"users": &UsersIngestor{
			gqlclient: g.gqlclient,
			db:        g.db,
			data:      &UsersData{},
			session:   g.session,
		},
		"repos": &ReposIngestor{
			gqlclient: g.gqlclient,
			db:        g.db,
			data:      &ReposData{},
			session:   g.session,
		},
		"organizationsecrets": &OrganizationSecretsIngestor{
			restclient: g.restclient,
			db:         g.db,
			data:       &OrganizationSecretsData{},
			session:    g.session,
		},
	}

	for _, name := range orgIngestorOrderedKeys {
		if !sliceContains(targetIngestors, name) {
			continue
		}
		log.Infof("Running org ingestor %s", name)
		orgIngestors[name].Sync()
	}

	// teamIngestors query at a specific team level
	teamRecords := g.db.Run(
		`MATCH (t:Team{session:$session}) RETURN t.slug as teamSlug`,
		map[string]interface{}{"session": g.session},
	)
	for teamRecords.Next() {
		teamSlug, _ := teamRecords.Record().Get("teamSlug")

		teamIngestors := map[string]Ingestor{
			"teamrepos": &TeamReposIngestor{
				gqlclient: g.gqlclient,
				db:        g.db,
				data:      &TeamReposData{},
				teamSlug:  teamSlug.(string),
				session:   g.session,
			},
			"teammembers": &TeamMembersIngestor{
				gqlclient: g.gqlclient,
				db:        g.db,
				data:      &TeamMembersData{},
				teamSlug:  teamSlug.(string),
				session:   g.session,
			},
		}

		for name, ingestor := range teamIngestors {
			if !sliceContains(targetIngestors, name) {
				continue
			}
			log.Infof("Running team ingestor %s on team %s", name, teamSlug)
			ingestor.Sync()
		}
	}

	g.runRepoIngestors(targetIngestors)
}

// Runs repository ingestors if they're in targetIngestors. Repository ingestors operate at a
// specific repo level
func (g *GitHub) runRepoIngestors(targetIngestors []string) {
	repoRecords := g.db.Run(`
		MATCH (r:Repository) RETURN r.name as repoName, r.databaseId as repoId
		`,
		map[string]interface{}{"session": g.session},
	)
	for repoRecords.Next() {
		repoName, _ := repoRecords.Record().Get("repoName")
		repoId, _ := repoRecords.Record().Get("repoId")

		repoIngestors := map[string]Ingestor{
			"repowebhooks": &RepoWebhooksIngestor{
				restclient: g.restclient,
				db:         g.db,
				repoName:   repoName.(string),
				session:    g.session,
			},
			"environments": &EnvironmentsIngestor{
				restclient: g.restclient,
				db:         g.db,
				repoName:   repoName.(string),
				session:    g.session,
			},
		}

		for name, ingestor := range repoIngestors {
			if !sliceContains(targetIngestors, name) {
				continue
			}
			log.Infof("Running repo ingestor %s on repo %s", name, repoName)
			ingestor.Sync()
		}

		g.runRepoEnvironmentIngestors(targetIngestors, repoName.(string), repoId.(int64))
	}
}

// Runs repository environment level ingestors on repositories if they're in the targetIngestors.
func (g *GitHub) runRepoEnvironmentIngestors(
	targetIngestors []string,
	repoName string,
	repoId int64,
) {
	repoEnvironments := g.db.Run(`
		MATCH (:Repository{session:$session, name:$repoName})-->(e:Environment{session:$session})
		RETURN e.name as envName
		`,
		map[string]interface{}{"session": g.session, "repoName": repoName},
	)

	for repoEnvironments.Next() {
		envName, _ := repoEnvironments.Record().Get("envName")
		envIngestors := map[string]Ingestor{
			"environmentsecrets": &EnvironmentSecretsIngestor{
				restclient: g.restclient,
				db:         g.db,
				repoId:     repoId,
				envName:    envName.(string),
				session:    g.session,
			},
		}

		for name, ingestor := range envIngestors {
			if !sliceContains(targetIngestors, name) {
				continue
			}
			log.Infof(
				"Running environment ingestor %s on repo %s for env %s",
				name,
				repoName,
				envName,
			)
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
