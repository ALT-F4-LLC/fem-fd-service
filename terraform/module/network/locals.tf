locals {
  subnets = {
    "database"    = 6,
    "elasticache" = 6,
    "intra"       = 5,
    "private"     = 3,
    "public"      = 5,
  }
}
