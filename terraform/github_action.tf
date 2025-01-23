#--------------------------------------------#
# Using locals instead of hard-coding strings
#--------------------------------------------#
locals {
  repository_default_branch_name         = "main"
  iam_role_name_apply                    = "gh-tf-apply-${substr(local.github_repository, 0, 64 - length("gh-tf-apply-"))}"
  iam_role_name_plan                     = "gh-tf-plan-${substr(local.github_repository, 0, 64 - length("gh-tf-apply-"))}"
  iam_policy_apply                       = "arn:aws:iam::aws:policy/AdministratorAccess"
  iam_policy_plan                        = "arn:aws:iam::aws:policy/ReadOnlyAccess"
  aws_ssm_name_github_token              = "/cicd/github_token"
  github_env_var_name_iam_role_plan_arn  = "AWS_IAM_ROLE_PLAN"
  github_env_var_name_iam_role_apply_arn = "AWS_IAM_ROLE_APPLY"
  github_env_var_name_aws_region         = "AWS_REGION"
  github_env_var_name_terraform_version  = "TF_VERSION"
  github_env_var_name_github_token       = "GH_TOKEN"
  github_actions_terraform_version       = "v1.10.3"

  # https://github.blog/changelog/2023-06-27-github-actions-update-on-oidc-integration-with-aws/
  github_cert_thumbprint = [
    "6938fd4d98bab03faadb97b34396831e3780aea1",
    "1c58a3a8518e8759bf075b76b750d4f2df264fcd"
  ]
}


provider "github" {
  owner = local.github_organization
  token = data.aws_ssm_parameter.github_token.value
}

# # Set up access from GitHub into the account. The thumbprint for GitHub
# # certificate can be used from the post 
# # https://github.blog/changelog/2022-01-13-github-actions-update-on-oidc-based-deployments-to-aws/
# # or generated. 
resource "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"

  thumbprint_list = local.github_cert_thumbprint
  tags            = local.tags
  client_id_list  = ["sts.amazonaws.com"]
}

#------------------------------------------------------------#
# IAM Role used to apply changes.
# Defaults to policy/AdministratorAccess, 
# but can be overridden to a custom policy
# by setting var.override_iam_policy_administrator_access_arn
#------------------------------------------------------------#

data "aws_iam_policy_document" "github_actions_write_assume_role_policy" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }

    # Condition to limit to default AWS OIDC audience
    # see: https://github.com/aws-actions/configure-aws-credentials?tab=readme-ov-file#oidc-audience
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    # Condition to limit to commits to the main branch
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:${local.github_organization}/${local.github_repository}:*"
      ]
    }
  }
}

# Role to allow GitHub actions to use this AWS account
resource "aws_iam_role" "github_actions_apply" {
  name               = local.iam_role_name_apply
  assume_role_policy = data.aws_iam_policy_document.github_actions_write_assume_role_policy.json
  tags               = local.tags
}

# Allow GitHub actions to create infrastructure
resource "aws_iam_role_policy_attachment" "github_actions_apply_policy" {
  role       = aws_iam_role.github_actions_apply.name
  policy_arn = local.iam_policy_apply
}

# Attach the state lock table access policy
resource "aws_iam_role_policy_attachment" "github_actions_apply_state_lock_policy" {
  role       = aws_iam_role.github_actions_apply.name
  policy_arn = module.aws-tf-kickstart.state_file_iam_policy_arn
}

#------------------------------------------------------------#
# IAM Role used to plan changes.
# Defaults to policy/ReadOnly, 
# but can be overridden to a custom policy
# by setting var.override_iam_policy_read_only_arn
#------------------------------------------------------------#

data "aws_iam_policy_document" "github_actions_read_assume_role_policy" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }

    # Condition to limit to default AWS OIDC audience
    # see: https://github.com/aws-actions/configure-aws-credentials?tab=readme-ov-file#oidc-audience
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    # Condition to limit to pull requests
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:${local.github_organization}/${local.github_repository}:pull_request",
        "repo:${local.github_organization}/${local.github_repository}:ref/pull/*",
        "repo:${local.github_organization}/${local.github_repository}:ref:refs/heads/${local.repository_default_branch_name}"
      ]
    }
    # # Condition to limit to pull requests targeting 'main' branch
    # condition {
    #   test     = "StringEquals"
    #   variable = "token.actions.githubusercontent.com:ref"
    #   values = [
    #     "refs/heads/${var.repository_default_branch_name}" # Only allow for PRs targeting the 'main' branch
    #   ]
    # }
  }
}

# Role to allow GitHub actions to use this AWS account to run terraform plan
resource "aws_iam_role" "github_actions_plan" {
  name               = local.iam_role_name_plan
  assume_role_policy = data.aws_iam_policy_document.github_actions_read_assume_role_policy.json
  tags               = local.tags
}

# Allow GitHub actions to create infrastructure
resource "aws_iam_role_policy_attachment" "github_actions_plan_policy" {
  role       = aws_iam_role.github_actions_plan.name
  policy_arn = local.iam_policy_plan
}

# Attach the state lock table access policy
resource "aws_iam_role_policy_attachment" "github_actions_plan_state_lock_policy" {
  role       = aws_iam_role.github_actions_plan.name
  policy_arn = module.aws-tf-kickstart.state_file_iam_policy_arn
}

#---------------------------------------------------------#
# Create the GitHub Actions workflow file in the code repo
#---------------------------------------------------------#

# resource "local_file" "github_actions_cicd_workflow" {
#   filename = "${path.root}/../.github/workflows/${local.github_terraform_workflow_file}"
#   content = templatefile("${path.module}/templates/github_actions_workflow.yml.tmpl", {
#     terraform_source_dir = local.terraform_source_dir
#     aws_region           = var.aws_region
#     github_organization  = local.github_organization
#     github_repository    = local.github_repository
#   })
# }

data "aws_ssm_parameter" "github_token" {
  name = local.aws_ssm_name_github_token
}

resource "github_actions_secret" "github_cicd_token" {
  repository  = local.github_repository
  secret_name = local.github_env_var_name_github_token

  # You can replace this with encrypted_value - this requires 
  # encrypting the value and storing the encrypted string in SSM,
  # see https://docs.github.com/en/rest/guides/encrypting-secrets-for-the-rest-api
  plaintext_value = data.aws_ssm_parameter.github_token.value
}

resource "github_actions_variable" "tf_version" {
  repository    = local.github_repository
  variable_name = local.github_env_var_name_terraform_version
  value         = local.github_actions_terraform_version
}

resource "github_actions_secret" "iam_policy_apply_changes_name" {
  repository      = local.github_repository
  secret_name     = local.github_env_var_name_iam_role_apply_arn
  plaintext_value = aws_iam_role.github_actions_apply.arn
}

resource "github_actions_secret" "iam_role_plan_changes_name" {
  repository      = local.github_repository
  secret_name     = local.github_env_var_name_iam_role_plan_arn
  plaintext_value = aws_iam_role.github_actions_plan.arn
}

resource "github_actions_variable" "aws_region" {
  repository    = local.github_repository
  variable_name = local.github_env_var_name_aws_region
  value         = local.aws_region
}

