terraform {
  backend "s3" {
    bucket = "terraform-fem-fd-service-preview"
    key    = "terraform.tfstate"
    region = "us-west-2"
  }
}
