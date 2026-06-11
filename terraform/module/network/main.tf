module "subnets" {
  source  = "hashicorp/subnets/cidr"
  version = "1.0.0"

  base_cidr_block = var.cidr

  networks = flatten([
    for k, v in local.subnets : [
      for az in var.availability_zones : {
        name     = "${k}-${az}"
        new_bits = v
      }
    ]
  ])
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.21.0"

  azs                    = var.availability_zones
  cidr                   = var.cidr
  database_subnets       = [for az in var.availability_zones : module.subnets.network_cidr_blocks["database-${az}"]]
  elasticache_subnets    = [for az in var.availability_zones : module.subnets.network_cidr_blocks["elasticache-${az}"]]
  enable_nat_gateway     = true
  intra_subnets          = [for az in var.availability_zones : module.subnets.network_cidr_blocks["intra-${az}"]]
  name                   = var.name
  one_nat_gateway_per_az = false
  private_subnets        = [for az in var.availability_zones : module.subnets.network_cidr_blocks["private-${az}"]]
  public_subnets         = [for az in var.availability_zones : module.subnets.network_cidr_blocks["public-${az}"]]
  single_nat_gateway     = true

  default_security_group_ingress = [
    {
      self = true
    }
  ]
}
