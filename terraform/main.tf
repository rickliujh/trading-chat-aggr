locals {
  proj_name           = "kickstart_gogrpc"
  env                 = "dev"
  github_organization = "rickliujh"
  github_repository   = "kickstart-gogrpc"
  aws_region          = "us-east-1"
  tags = {
    app = local.proj_name
    env = local.env
  }
}

module "aws-tf-kickstart" {
  source = "./modules/kickstart"

  state_file_aws_region          = "us-east-1"
  state_file_bucket_name         = "tf-state-gogrpc"
  override_state_lock_table_name = "tf-state-lock-gogrpc"
  override_aws_tags              = local.tags
  tf_additional_providers = [
    {
      name             = "github"
      provider_source  = "integrations/github"
      provider_version = "6.0"
    }
  ]
}

