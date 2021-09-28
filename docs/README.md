# Docs

## Overview

In addition to mapping relationships between your users, teams and repositories, GitOops maps relationships between those and environment variables in your CI/CD systems.

The following CI/CD systems are currently supported:

- GitHub Actions
- CircleCI

On top of this, GitOops ingests CI/CD configuration files from repositories for other popular CI/CD systems, enabling less refined queries on those.

Finally, GitOops will also map webhooks and status checks from commits to a repository's pull requests and default branch. These allow you to find integrations that are typically configured server-side (e.g. AWS CodeBuild).

## Content

- [Install, Build & Run](run.md)
- [Examples](examples.md)
- [Schema](schema.md)
- [Blog post](blog.md)
