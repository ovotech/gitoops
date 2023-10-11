# Install, Build & Run

You can build GitOops yourself or use our binaries.

You may also consider using it as a package if you want to run some custom ingestion.

## Install

Download the latest release for your OS from the [releases page](https://github.com/ovotech/gitoops/releases/latest) or:

```
$ export OS=linux # or macos/windows
$ curl -Lso gitoops "https://github.com/ovotech/gitoops/releases/latest/download/gitoops-$OS"
```

## Build

```
$ go version
go version go1.16.6 linux/amd64
$ git clone git@github.com:ovotech/gitoops.git
$ cd gitoops
$ make
$ ./gitoops
Usage: ./gitoops [SUBCOMMAND] [OPTIONS]...
Available subcommands:
	circleci
	enrich
	github
```

## CLI

### Database

You will need a Bolt-compatible database. We provide a `docker-compose` file for Neo4j.

IMPORTANT: This sets up an _unauthenticated_ Neo4j instance listening on localhost. You're recommended to set a password.

```
$ docker-compose -f docker-compose.yml up -d
```

### Ingest GitHub data

GitOops uses a Personal Access Token (PAT) to ingest GitHub data. You will need the `read:org` and `repo` (`Full control of private repositories`) scopes.

To get full coverage you should use an organization owner PAT. You can use an organization member PAT but you will get only partial coverage.

```
$ gitoops github                              \
          -debug                              \
          -organization fakenews              \
          -neo4j-password $NEO4J_PASSWORD     \
          -neo4j-uri="neo4j://localhost:7687" \
          -token $GITHUB_TOKEN                \
          -ingestor default                   \
          -ingestor secrets                   \
          -session helloworld
```

Most parameters should be self-explanatory.

Please check `gitoops github -h` for more information on the `-ingestor`.

The `session` is just a unique identifier for this run of the ingestor. You can use this to remove old nodes and relationships that are no longer relevant (by removing any nodes and relationships that don't have the latest session identifier from your database).

#### Note on Rate Limits

If you are targeting a large GitHub organization, you may encounter rate limits. If this happens you can use the `-ingestor` flags to limit the information you are ingesting at a time.

The following ingestors need to run first and in this particular order:

- Organizations
- Teams
- Users
- Repos

Order doesn't matter for other ingestors.

#### GitHub Enterprise Server

If you are targeting a self-hosted GitHub Enterprise Server, you will want to set the `-github-rest-url` and `-github-graphql-url` parameters. These default to the GitHub cloud URLs.

### Ingest CircleCI data

Unfortunately, the documented CircleCI REST API doesn't give everything we want. Luckily there's a "hidden" GraphQL API we can access with a cookie. With your browser, navigate to the CircleCI web UI and fetch your `ring-session` cookie. You should be able to find this in a request to the `graphql-unstable` endpoint when loading some pages.

```
$ export CIRCLECI_COOKIE=RING_SESSION_COOKIE_VALUE
$ gitoops circleci                            \
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
$ gitoops enrich                             \
          -debug                             \
          -organization fakenews             \
          -session helloworld                \
          -neo4j-password $NEO4J_PASSWORD    \
          -neo4j-uri="neo4j://localhost:7687"
```

## Package

TODO
