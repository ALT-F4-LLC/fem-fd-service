module "staging" {
  source = "./module/environment"

  bastion_ingress = local.bastion_ingress
  name            = "staging"
}

module "prod" {
  source = "./module/environment"

  bastion_ingress = local.bastion_ingress
  name            = "prod"
}
