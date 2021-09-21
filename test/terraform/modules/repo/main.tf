// we need to redefine the provider here or it doesn't seem to pick up the fact we want to operate 
// on an org and instead tries to operate on calling user
terraform {
  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 4.0"
    }
    circleci = {
      source = "TomTucka/circleci"
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

resource "github_branch" "trigger" {
  for_each   = fileset(var.files_path, "**")
  repository = var.name
  branch     = var.branch

  depends_on = [
    github_repository.repo,
    github_repository_file.repo
  ]
}

resource "github_repository_file" "trigger" {
  repository          = github_repository.repo.name
  branch              = var.branch
  file                = "trigger"
  content             = "trigger pipeline"
  commit_message      = "trigger"
  commit_author       = "Terraform User"
  commit_email        = "terraform@example.com"
  overwrite_on_create = true

  depends_on = [
    github_branch.trigger
  ]
}

resource "circleci_project" "repo" {
  count = var.circleci ? 1 : 0
  name  = var.name

  depends_on = [
    github_repository.repo,
    github_repository_file.repo,
    github_branch.trigger
  ]
}

resource "time_sleep" "wait_for_cirlceci" {
  create_duration = "30s"

  depends_on = [
    circleci_project.repo
  ]

}

resource "github_repository_pull_request" "pr" {
  base_repository = var.name
  base_ref        = "main"
  head_ref        = var.branch
  title           = "trigger pipeline"
  body            = "this is just a demo pipeline run"

  depends_on = [
    time_sleep.wait_for_cirlceci
  ]
}