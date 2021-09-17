// we need to redefine the provider here or it doesn't seem to pick up the fact we want to operate 
// on an org and instead tries to operate on calling user
terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 4.0"
    }
  }
}

resource "github_repository" "repo" {
  name      = var.name
  auto_init = true
}

resource "github_repository_file" "repo" {
  for_each = fileset(var.files_path, "**")

  repository          = github_repository.repo.name
  branch              = "main"
  file                = each.value
  content             = file("${var.files_path}/${each.value}")
  commit_message      = "init"
  commit_author       = "Terraform User"
  commit_email        = "terraform@example.com"
  overwrite_on_create = true
}

resource "github_branch" "development" {
  for_each   = fileset(var.files_path, "**")
  repository = var.name
  branch     = "trigger"
}