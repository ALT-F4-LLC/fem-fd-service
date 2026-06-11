resource "aws_ecs_cluster" "this" {
  name = var.name

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

resource "aws_iam_role" "this" {
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
  name               = var.name
}

resource "aws_iam_role_policy_attachment" "service_role" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role"
  role       = aws_iam_role.this.name
}

resource "aws_iam_instance_profile" "this" {
  name = var.name
  role = aws_iam_role.this.name
}

resource "tls_private_key" "this" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "this" {
  key_name   = "${var.name}-cluster"
  public_key = tls_private_key.this.public_key_openssh
}

module "private-key" {
  source  = "terraform-aws-modules/ssm-parameter/aws"
  version = "1.1.2"

  name        = "/${var.name}/cluster/private-key"
  secure_type = true
  value       = tls_private_key.this.private_key_pem
}

resource "aws_launch_template" "this" {
  for_each = { for provider_name, provider in var.capacity_providers : provider_name => provider }

  image_id      = jsondecode(data.aws_ssm_parameter.ecs_optimized_ami.value)["image_id"]
  instance_type = each.value.instance_type
  key_name      = aws_key_pair.this.key_name
  name_prefix   = "${var.name}-${each.key}-"

  block_device_mappings {
    device_name = "/dev/xvda"

    ebs {
      delete_on_termination = true
      volume_size           = 50
      volume_type           = "gp2"
    }
  }

  iam_instance_profile {
    name = aws_iam_instance_profile.this.name
  }

  dynamic "instance_market_options" {
    for_each = each.value.market_type == "spot" ? [1] : []

    content {
      market_type = each.value.market_type
    }
  }

  network_interfaces {
    associate_public_ip_address = false
    security_groups             = var.security_groups
  }

  user_data = base64encode(templatefile("${path.module}/user_data.tpl", {
    cluster_name = var.name
  }))
}

resource "aws_autoscaling_group" "this" {
  for_each = { for provider_name, provider in var.capacity_providers : provider_name => provider }

  desired_capacity    = 1
  max_size            = 5
  min_size            = 1
  name_prefix         = "${var.name}-${each.key}-"
  vpc_zone_identifier = var.subnets

  launch_template {
    id      = aws_launch_template.this[each.key].id
    version = "$Latest"
  }

  instance_refresh {
    strategy = "Rolling"
    triggers = ["tag"]

    preferences {
      min_healthy_percentage = 50
    }
  }

  tag {
    key                 = "AmazonECSManaged"
    propagate_at_launch = true
    value               = "true"
  }

  tag {
    key                 = "Name"
    propagate_at_launch = true
    value               = var.name
  }
}

resource "aws_autoscaling_policy" "this" {
  for_each = { for provider_name, provider in var.capacity_providers : provider_name => provider }

  autoscaling_group_name = aws_autoscaling_group.this[each.key].name
  name                   = "${var.name}-${each.key}-cpu-target-tracking"
  policy_type            = "TargetTrackingScaling"

  target_tracking_configuration {
    predefined_metric_specification {
      predefined_metric_type = "ASGAverageCPUUtilization"
    }

    target_value = 50
  }
}

resource "aws_ecs_capacity_provider" "this" {
  for_each = { for provider_name, provider in var.capacity_providers : provider_name => provider }

  name = "${var.name}-${each.key}"

  auto_scaling_group_provider {
    auto_scaling_group_arn = aws_autoscaling_group.this[each.key].arn

    managed_scaling {
      status = "DISABLED"
    }
  }

  depends_on = [aws_security_group.load_balancer]
}

resource "aws_ecs_cluster_capacity_providers" "this" {
  capacity_providers = [for provider_name, provider in var.capacity_providers : aws_ecs_capacity_provider.this[provider_name].name]
  cluster_name       = var.name
}
