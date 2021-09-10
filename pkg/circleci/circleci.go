package circleci

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/ovotech/gitoops/pkg/database"
	log "github.com/sirupsen/logrus"
)

type CircleCI struct {
	gqlclient    *GraphQLClient
	restclient   *RESTClient
	db           *database.Database
	organization string
	session      string
}

func GetCircleCI(db *database.Database, organization, cookie, session string) *CircleCI {
	// Make sure cookie value is URL encoded by checking it doesn't have characters we'd expect to
	// be encoded.
	re := regexp.MustCompile(`(\+|\/|-|=| )`)
	if re.FindString(cookie) != "" {
		log.Debug("Cookie doesn't appear to be URL encoded, we will encode it.")
		cookie = url.QueryEscape(cookie)
	}

	cci := &CircleCI{
		gqlclient: &GraphQLClient{
			client: &http.Client{},
			cookie: cookie,
		},
		restclient: &RESTClient{
			client: &http.Client{},
			cookie: cookie,
		},
		db:           db,
		organization: organization,
		session:      session,
	}

	return cci
}

func (cci *CircleCI) Sync() {
	log.Info("Running OrganizationIngestor")
	oi := OrganizationIngestor{
		gqlclient: cci.gqlclient,
		// db:           cci.db,
		data:         &OrganizationData{},
		organization: cci.organization,
		session:      cci.session,
	}
	organizationId := oi.GetOrganizationId()

	log.Info("Running ContextsIngestor")
	ci := ContextsIngestor{
		gqlclient:      cci.gqlclient,
		db:             cci.db,
		data:           &ContextsData{},
		organizationId: organizationId,
		session:        cci.session,
	}
	ci.Sync()

	contextRecords := cci.db.Run(
		`MATCH (c:CircleCIContext{session:$session}) RETURN c.id as contextId, c.name as contextName`,
		map[string]interface{}{"session": cci.session},
	)
	for contextRecords.Next() {
		contextId, _ := contextRecords.Record().Get("contextId")
		contextName, _ := contextRecords.Record().Get("contextName")

		log.Infof("Running ContextsEnvVarsIngestor on context %s (%s)", contextName, contextId)
		cevi := ContextEnVarsIngestor{
			gqlclient: cci.gqlclient,
			db:        cci.db,
			data:      &ContextEnvVarsData{},
			contextId: contextId.(string),
			session:   cci.session,
		}
		cevi.Sync()
	}

	// repoIngestors query at a specific repo level
	repoRecords := cci.db.Run(
		`MATCH (r:Repository{session:$session}) RETURN r.name as repoName`,
		map[string]interface{}{"session": cci.session},
	)
	for repoRecords.Next() {
		repoName, _ := repoRecords.Record().Get("repoName")

		log.Infof("Running ProjectIngestor on repo %s", repoName)
		pi := ProjectIngestor{
			restclient:   cci.restclient,
			db:           cci.db,
			data:         &ProjectData{},
			organization: cci.organization,
			repoName:     repoName.(string),
			session:      cci.session,
		}
		pi.Sync()
	}

	// queries existing CircleCIProjects
	projectRecords := cci.db.Run(
		`MATCH (p:CircleCIProject{session:$session}) RETURN p.repository as projectName`,
		map[string]interface{}{"session": cci.session},
	)
	for projectRecords.Next() {
		projectName, _ := projectRecords.Record().Get("projectName")

		log.Infof("Running ProjectEnvVarsIngestor on projet %s", projectName)
		pevi := ProjectEnvVarsIngestor{
			restclient:   cci.restclient,
			db:           cci.db,
			data:         &ProjectEnvVarsData{},
			organization: cci.organization,
			projectName:  projectName.(string),
			session:      cci.session,
		}
		pevi.Sync()
	}
}
