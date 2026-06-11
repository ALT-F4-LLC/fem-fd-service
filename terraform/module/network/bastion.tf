resource "tls_private_key" "bastion" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "bastion" {
  key_name   = "${var.name}-bastion"
  public_key = tls_private_key.bastion.public_key_openssh
}

module "bastion-private-key" {
  source  = "terraform-aws-modules/ssm-parameter/aws"
  version = "1.1.2"

  name        = "/${var.name}/bastion/private-key"
  secure_type = true
  value       = tls_private_key.bastion.private_key_pem
}

module "bastion" {
  source  = "terraform-aws-modules/ec2-instance/aws"
  version = "5.8.0"

  associate_public_ip_address = true
  instance_type               = "t3a.micro"
  key_name                    = aws_key_pair.bastion.key_name
  monitoring                  = true
  name                        = "${var.name}-bastion"
  subnet_id                   = module.vpc.public_subnets[0]

  vpc_security_group_ids = [
    module.security_group_bastion.security_group_id,
    module.security_group_private.security_group_id,
  ]
}
