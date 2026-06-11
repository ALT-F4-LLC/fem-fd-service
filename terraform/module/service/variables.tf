variable "capacity_provider" {
  type = string
}

variable "cluster_id" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "config" {
  default = {}
  type    = map(string)
}

variable "cpu" {
  default = 256
  type    = number
}

variable "image_registry" {
  default = "public.ecr.aws"
  type    = string
}

variable "image_repository" {
  default = "nginx/nginx"
  type    = string
}

variable "image_tag" {
  default = "alpine"
  type    = string
}

variable "listener_arn" {
  type = string
}

variable "log_retention" {
  default = 7
  type    = number
}

variable "memory" {
  default = 512
  type    = number
}

variable "name" {
  type = string
}

variable "paths" {
  type = list(string)
}

variable "port" {
  default = 80
  type    = number
}

variable "secrets" {
  default = []
  type    = list(string)
}

variable "vpc_id" {
  type = string
}
