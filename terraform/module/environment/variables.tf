variable "bastion_ingress" {
  default     = []
  description = "CIDR blocks for bastion ingress"
  type        = list(string)
}

variable "name" {
  description = "Name of the cloud environment"
  type        = string
}
