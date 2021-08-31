package enrich

import (
	"regexp"
	"strings"

	"github.com/ovotech/gitoops/pkg/database"
)

type Enricher struct {
	db           *database.Database
	organization string
}

func GetEnricher(db *database.Database, organization string) *Enricher {
	return &Enricher{
		db:           db,
		organization: organization,
	}
}

func (e *Enricher) Enrich() {
	e.extractEnvVarsFromCIFiles()
	e.tagCIFiles()
}

func (e *Enricher) extractEnvVarsFromCIFiles() {
	r, _ := regexp.Compile("[A-Z_]{2,}")

	records := e.db.Run(`
	MATCH (f:File) RETURN f.text as text, f.id as id
	`, map[string]interface{}{})

	for records.Next() {
		id, _ := records.Record().Get("id")
		text, _ := records.Record().Get("text")
		matches := r.FindAllString(text.(string), -1)
		envVars := removeDuplicateStringsFromSlice(matches)

		e.db.Run(`
		MATCH (f:File{id: $id})
		SET f.env = $envVars
		`, map[string]interface{}{"id": id, "envVars": envVars})
	}
}

func removeDuplicateStringsFromSlice(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func (e *Enricher) tagCIFiles() {
	keyTerms := map[string][]string{
		"aws":         {"aws", "ecr"},
		"gcp":         {"gcp", "gcr", "gcloud"},
		"dockerhub":   {"dockerhub"},
		"artifactory": {"artifactory"},
		"terraform":   {"tf", "terraform"},
		"bintray":     {"bintray"},
		"kafka":       {"kafka", "aiven"},
	}

	records := e.db.Run(`
	MATCH (f:File) RETURN f.text as text, f.id as id
	`, map[string]interface{}{})

	for records.Next() {
		tags := []string{}
		id, _ := records.Record().Get("id")
		text, _ := records.Record().Get("text")
		lowerText := strings.ToLower(text.(string))

		for tag, terms := range keyTerms {
			for _, term := range terms {
				if strings.Contains(lowerText, term) {
					tags = append(tags, tag)
					break
				}
			}
		}

		e.db.Run(`
		MATCH (f:File{id: $id})
		SET f.tags = $tags
		`, map[string]interface{}{"id": id, "tags": tags})
	}
}
