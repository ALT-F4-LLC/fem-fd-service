variable "name" {
  description = "The name of the database"
  type        = string
}

variable "security_groups" {
  description = "The security groups to deploy the database in"
  type        = list(string)
}

variable "subnets" {
  description = "The subnets to deploy the database in"
  type        = list(string)
}

variable "vpc_name" {
  description = "The name of the VPC"
  type        = string
}
