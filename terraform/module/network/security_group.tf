module "security_group_bastion" {
  source  = "terraform-aws-modules/security-group/aws"
  version = "5.3.0"

  description        = "Security group for bastion server"
  egress_cidr_blocks = ["0.0.0.0/0"]
  name               = "${var.name}-bastion"
  vpc_id             = module.vpc.vpc_id

  ingress_with_cidr_blocks = [
    for cidr in var.bastion_ingress : {
      cidr_blocks = cidr,
      rule        = "ssh-tcp",
    }
  ]

  ingress_with_self = [
    {
      rule = "all-all"
    },
  ]

  egress_with_cidr_blocks = [
    {
      rule = "all-all"
    }
  ]
}

module "security_group_db" {
  source  = "terraform-aws-modules/security-group/aws"
  version = "5.3.0"

  description        = "Security group for db subnet"
  egress_cidr_blocks = ["0.0.0.0/0"]
  name               = "${var.name}-db"
  vpc_id             = module.vpc.vpc_id

  ingress_with_self = [
    {
      rule = "all-all"
    }
  ]

  egress_with_cidr_blocks = [
    {
      rule = "all-all"
    }
  ]
}

module "security_group_elasticache" {
  source  = "terraform-aws-modules/security-group/aws"
  version = "5.3.0"

  description        = "Security group for elasticache subnet"
  egress_cidr_blocks = ["0.0.0.0/0"]
  name               = "${var.name}-elasticache"
  vpc_id             = module.vpc.vpc_id

  ingress_with_self = [
    {
      rule = "all-all"
    }
  ]

  egress_with_cidr_blocks = [
    {
      rule = "all-all"
    }
  ]
}

module "security_group_private" {
  source  = "terraform-aws-modules/security-group/aws"
  version = "5.3.0"

  description        = "Security group for private subnet"
  egress_cidr_blocks = ["0.0.0.0/0"]
  name               = "${var.name}-private"
  vpc_id             = module.vpc.vpc_id

  ingress_with_self = [
    {
      rule = "all-all"
    }
  ]

  egress_with_cidr_blocks = [
    {
      rule = "all-all"
    }
  ]
}

resource "aws_vpc_security_group_ingress_rule" "db_allow_private" {
  description                  = "Allow private subnet to access db"
  from_port                    = 5432
  ip_protocol                  = "tcp"
  referenced_security_group_id = module.security_group_private.security_group_id
  security_group_id            = module.security_group_db.security_group_id
  to_port                      = 5432
}

resource "aws_vpc_security_group_ingress_rule" "elasticache_allow_private" {
  description                  = "Allow private subnet to access elasticache"
  from_port                    = 6379
  ip_protocol                  = "tcp"
  referenced_security_group_id = module.security_group_private.security_group_id
  security_group_id            = module.security_group_elasticache.security_group_id
  to_port                      = 6379
}
