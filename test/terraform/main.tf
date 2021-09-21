terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 4.0"
    }
    // two different third-party providers from randomers (we're not doing any sensitive with these)
    // one supports contexts, the other supports projects
    circlecicontexts = {
      source = "mrolla/circleci"
    }
    circleciprojects = {
      source = "TomTucka/circleci"
    }
  }
}

provider "github" {
  owner = var.org_name
}

// there seems to be a bug with the github provider where the org configuration is not picked up in 
// child modules. so we use another aliased provider to pass to modules
provider "github" {
  alias = "github"
  owner = var.org_name
}

provider "circlecicontexts" {
  vcs_type     = "github"
  organization = var.org_name
}

provider "circleciprojects" {
  organization = var.org_name
  vcs_type     = "github"
}
