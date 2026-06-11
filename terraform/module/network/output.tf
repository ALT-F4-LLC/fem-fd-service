output "database_security_group" {
  value = module.security_group_db.security_group_id
}

output "database_subnets" {
  value = module.vpc.database_subnets
}

output "private_security_group" {
  value = module.security_group_private.security_group_id
}

output "private_subnets" {
  value = module.vpc.private_subnets
}

output "vpc_id" {
  value = module.vpc.vpc_id
}

output "vpc_name" {
  value = module.vpc.name
}
