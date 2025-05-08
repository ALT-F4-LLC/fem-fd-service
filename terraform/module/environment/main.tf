module "network" {
  source = "../network"

  availability_zones = ["us-west-2a", "us-west-2b", "us-west-2c"]
  bastion_ingress    = var.bastion_ingress
  cidr               = "10.0.0.0/16"
  name               = var.name
}

module "database" {
  source = "../database"

  security_groups = [module.network.database_security_group]
  subnets         = module.network.database_subnets
  name            = var.name
  vpc_name        = module.network.vpc_name
}

module "cluster" {
  source = "../cluster"

  security_groups = [module.network.private_security_group]
  subnets         = module.network.private_subnets
  name            = var.name
  vpc_id          = module.network.vpc_id

  capacity_providers = {
    "spot" = {
      instance_type = "t3a.medium"
      market_type   = "spot"
    }
  }
}

module "service" {
  source = "../service"

  capacity_provider = "spot"
  cluster_id        = module.cluster.cluster_arn
  cluster_name      = var.name
  image_registry    = "${data.aws_caller_identity.this.account_id}.dkr.ecr.${data.aws_region.this.name}.amazonaws.com"
  image_repository  = "fem-fd-service"
  image_tag         = var.name
  listener_arn      = module.cluster.listener_arn
  name              = "service"
  paths             = ["/*"]
  port              = 8080
  vpc_id            = module.network.vpc_id

  config = {
    GOOGLE_REDIRECT_URL = "https://${module.cluster.distribution_domain}/auth/google/callback"
    GOOSE_DRIVER        = "postgres"
  }

  secrets = [
    "GOOGLE_CLIENT_ID",
    "GOOGLE_CLIENT_SECRET",
    "GOOSE_DBSTRING",
    "POSTGRES_URL",
  ]
}
