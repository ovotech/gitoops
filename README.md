# GitOops! 😱

> all paths lead to clouds

GitOops is a tool to help attackers and defenders identify lateral movement and privilege escalation paths in GitHub organizations by abusing CI/CD pipelines and GitHub access controls.

It works by ingesting security-relevant information from your GitHub organization and your CI/CD systems into a Bolt-compatible graph database, allowing you the query attack paths with openCypher (Neo4j, AWS Neptune...)

GitOops takes inspiration from tools like [Bloodhound](https://github.com/BloodHoundAD/BloodHound) and [Cartography](https://github.com/lyft/cartography).

_This project is still in early stages and substantial changes to the codebase and database schema are to be expected._

## Supported CI/CD Systems

In addition to mapping relationships between your users, teams and repositories, GitOops maps relationships between those and environment variables in your CI/CD systems.

The following CI/CD systems are currently supported:

- GitHub Actions
- CircleCI

On top of this, it ingests CI/CD configuration files from repositories for other popular CI/CD systems, enabling less refined queries on those.

Finally, GitOops will also map status checks from commits to a repository's pull requests and default branch, allowing you to find integrations that are typically configured server-side (e.g. AWS CodeBuild).

## Examples

- Show me all CI/CD environment variables my GitHub user has direct or indirect acces to

This query will return paths between your user and potential secrets, via any means. This will include CircleCI "All members" contexts (available to any GitHub user in your organization), team-restricted CircleCI contexts, repository-based CircleCI project environment variables and repository-based GitHub Actions environment variables.

```
MATCH p=(:User{login:"serain"})-[*..5]->(:EnvironmentVariable)
RETURN p
```

- I just compromised a GitHub user called `superbot`. What new repositories did I gain access to?

```
MATCH (b:User{login:"superbot"})-->(r:Repository)
WHERE NOT EXISTS((:User{login:'serain'})-->(r))
RETURN r.name
```

- Show me repositories running production `terraform plan` on pull requests

Production Terraform plans on unreviewed code are [a bad idea](https://alex.kaskaso.li/post/terraform-plan-rce). We attempt to find these by looking at the context values on pull requests' status checks, to get maximum coverage and account for CI/CD systems that may be configured server-side (e.g. AWS CodeBuild).

```
MATCH (r:Repository)-[:HAS_STATUS_CHECK{pull_request:TRUE}]->(s:StatusCheck)
WHERE s.context =~ "(?=.*(tf|terraform))(?=.*(?<!non)pro?d).*"
RETURN r
```

- Show me `AWS_SECRET_ACCESS_KEY` variables my user can access through `WRITE` access to repositories with CircleCI project environment variables

CircleCI doesn't support branch-level protections for secrets. The implication is that if you can open a PR against a repository, you can exfiltrate secrets from the CI/CD context. These could be production secrets.

Note that our query matches repositories that the user can access both directly and indirectly, through team memberships.

This query is broken down to illustrate the relationships GitOops builds up; the query could be written more succinctly.

```
MATCH p=(:User{login:"serain"})-[*..2]->(:Repository)-[:HAS_CI]->(:CircleCIProject)-[:EXPOSES_ENVIRONMENT_VARIABLE]->(v:EnvironmentVariable)
WHERE v.variable =~ ".*AWS.*SECRET.*"
RETURN p
```

- Show me repositories that many users have access to with potentially interesting secrets exposed in a pull request's CI/CD context

Here we're using the content of CI/CD configuration files to make educated guesses about interesting pipelines. This is less accurate that using our other relationships, but gives us coverage of unsupported CI/CD systems (as long as we pulled the configuration files).

```
MATCH (r:Repository)-[HAS_CI_CONFIGURATION]->(f:File{path: ".circleci/config.yml"})
WHERE any(x IN f.env WHERE x =~ ".*(AUTH|SECRET|TOKEN|PASS|PWD|CRED|KEY|PRD|PROD).*")
OR any(x IN f.tags WHERE x IN ["aws", "gcp", "terraform"])

WITH r
MATCH (u:User)-[*..2]->(r)-[HAS_STATUS_CHECK{pull_request:true}]->(s:StatusCheck)

WITH r, COUNT(DISTINCT u) AS userCount
WHERE userCount > 30
RETURN r
```

- Show me repositories with potential automated deployments that don't have branch protections

```
MATCH (r:Repository)-[HAS_CI_CONFIGURATION]->(f:File)
WHERE any(x IN f.env WHERE x =~ ".*(AUTH|SECRET|TOKEN|PASS|PWD|CRED|KEY|PRD|PROD).*")
OR any(x IN f.tags WHERE x IN ["aws", "gcp", "terraform"])
WITH r

MATCH (r)
WHERE NOT (r)-[:HAS_BRANCH_PROTECTION_RULE]->(:BranchProtectionRule)
RETURN r
```

- Do any external contributors have paths to sensitive secrets?

```
MATCH (u:User)
WHERE NOT (u)-[:IS_MEMBER_OF]->(:Organization{login:"fakenews"})
WITH u

MATCH p=(u)-[*..5]->(v:EnvironmentVariable)
WHERE v.variable =~ ".*(AUTH|SECRET|TOKEN|PASS|PWD|CRED|KEY|PRD|PROD).*"

RETURN p
```

- Show me who's still using Jenkins

```
MATCH p=(t:Team)-[:HAS_PERMISSION_ON]->(r:Repository{isArchived:FALSE})-[:HAS_CI_CONFIGURATION_FILE]->(f:File{path:"Jenkinsfile"})
RETURN p
```

## CLI

### Ingest GitHub data

```
$ cd cmd/
$ go run . github                              \
           -debug                              \
           -organization fakenews              \
           -neo4j-password $NEO4J_PASSWORD     \
           -neo4j-uri="neo4j://localhost:7687" \
           -token $GITHUB_TOKEN                \
           -ingestor default                   \
           -session helloworld
```

Most parameters should be self-explanatory. Note that the `session` is just a unique identifier for this run of the ingestor. You can use this to remove old nodes and relationships that are no longer relevant (by removing any nodes and relationships that don't have the latest session identifier from your database).

### Ingest CircleCI data

Unfortunately, the documented CircleCI REST API doesn't give everything we want. Luckily there's a "hidden" GraphQL API we can access with a cookie. With your browser, navigate to the CircleCI web UI and fetch your `ring-session` cookie. You should be able to find this in a request to the `graphql-unstable` endpoint when loading some pages.

```
$ export CIRCLECI_COOKIE=RING_SESSION_COOKIE_VALUE
$ go run . circleci                            \
           -debug                              \
           -organization fakenews              \
           -neo4j-password $NEO4J_PASSWORD     \
           -neo4j-uri="neo4j://localhost:7687" \
           -cookie=$CIRCLECI_COOKIE            \
           -session helloworld
```

### Data enrichment

We do some very crude "enriching" of data. After you've ingested GitHub proceed to:

```
$ go run . enrich                             \
           -debug                             \
           -organization fakenews             \
           -session helloworld                \
           -neo4j-password $NEO4J_PASSWORD    \
           -neo4j-uri="neo4j://localhost:7687"
```
