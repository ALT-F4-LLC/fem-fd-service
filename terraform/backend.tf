terraform {
  backend "s3" {
    bucket = "fem-fd-service-altf4"
    key    = "terraform.tfstate"
    region = "us-west-2"
  }
}
