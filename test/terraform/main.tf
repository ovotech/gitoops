terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 4.0"
    }
  }
}

# Configure the GitHub Provider
provider "github" {
  owner = var.org_name
}

// there seems to be a bug with the github provider where the org configuration is not picked up in 
// child modules. so we use another aliased provider to pass to modules
provider "github" {
  alias = "github"
  owner = var.org_name
}
