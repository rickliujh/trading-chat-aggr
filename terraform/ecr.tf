locals {
  ecr_repo_name                                  = "${local.proj_name}-${local.env}"
  github_env_var_name_ecr_repo_name              = "ECR_REPOSITORY"
  github_env_var_name_iam_role_ecr_push_only_arn = "AWS_IAM_ROLE_ECR"
}

resource "aws_ecr_repository" "repo" {
  name = local.ecr_repo_name

  tags = local.tags
}

data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "ecr_policies" {
  statement {
    sid    = "AllowEcrPullAllSubAccounts"
    effect = "Allow"
    principals {
      type        = "AWS"
      identifiers = ["*"]
    }
    actions = [
      "ecr:GetDownloadUrlForLayer",
      "ecr:BatchGetImage",
      "ecr:ListImages"
    ]
    condition {
      test     = "StringEquals"
      variable = "aws:PrincipalAccount"
      values   = ["${data.aws_caller_identity.current.account_id}"]
    }
  }
}

resource "aws_ecr_repository_policy" "ecr" {
  repository = aws_ecr_repository.repo.name
  policy     = data.aws_iam_policy_document.ecr_policies.json
}

data "aws_iam_policy_document" "github_actions_ecr_push_permissions" {
  statement {
    sid    = "AllowEcrPushOnlyGithubActions"
    effect = "Allow"
    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:CompleteLayerUpload",
      "ecr:InitiateLayerUpload",
      "ecr:PutImage",
      "ecr:UploadLayerPart"
    ]
    resources = [aws_ecr_repository.repo.arn]
  }
}

data "aws_iam_policy_document" "github_actions_ecr_login_permission" {
  statement {
    sid    = "AllowEcrLogin"
    effect = "Allow"
    actions = [
      "ecr:GetAuthorizationToken",
    ]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "github_actions_ecr_push_assume_role_policy" {
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

    # Condition to limit to current repo
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:${local.github_organization}/${local.github_repository}:*"
      ]
    }
  }
}

resource "aws_iam_policy" "github_actions_ecr_push_policy" {
  name   = "gh-tf-ecr-push-only-${substr(local.github_repository, 0, 64 - length("gh-tf-ecr-push-only-"))}"
  policy = data.aws_iam_policy_document.github_actions_ecr_push_permissions.json
  tags   = local.tags
}

# Role to allow GitHub actions to use this AWS account
resource "aws_iam_role" "github_actions_ecr_push" {
  name               = "gh-tf-erc-push-${substr(local.github_repository, 0, 64 - length("gh-tf-erc-push-"))}"
  assume_role_policy = data.aws_iam_policy_document.github_actions_ecr_push_assume_role_policy.json
  tags               = local.tags
}

resource "aws_iam_policy" "github_actions_ecr_login_policy" {
  name   = "gh-tf-ecr-login-only-${substr(local.github_repository, 0, 64 - length("gh-tf-ecr-push-only-"))}"
  policy = data.aws_iam_policy_document.github_actions_ecr_login_permission.json
  tags   = local.tags
}

# Allow GitHub actions to create infrastructure
resource "aws_iam_role_policy_attachment" "github_actions_ecr_push_policy" {
  role       = aws_iam_role.github_actions_ecr_push.name
  policy_arn = aws_iam_policy.github_actions_ecr_push_policy.arn
}

# Attach the state lock table access policy
resource "aws_iam_role_policy_attachment" "github_actions_ecr_push_state_lock_policy" {
  role       = aws_iam_role.github_actions_ecr_push.name
  policy_arn = module.aws-tf-kickstart.state_file_iam_policy_arn
}

resource "aws_iam_role_policy_attachment" "github_actions_ecr_login_policy" {
  role       = aws_iam_role.github_actions_ecr_push.name
  policy_arn = aws_iam_policy.github_actions_ecr_login_policy.arn
}

resource "github_actions_variable" "ecr_repo_name" {
  repository    = local.github_repository
  variable_name = local.github_env_var_name_ecr_repo_name
  value         = local.ecr_repo_name
}

resource "github_actions_secret" "iam_role_ecr_push_only" {
  repository      = local.github_repository
  secret_name     = local.github_env_var_name_iam_role_ecr_push_only_arn
  plaintext_value = aws_iam_role.github_actions_ecr_push.arn
}
