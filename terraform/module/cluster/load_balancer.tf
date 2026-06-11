resource "aws_security_group" "load_balancer" {
  name_prefix = var.name
  vpc_id      = var.vpc_id
}

resource "aws_vpc_security_group_ingress_rule" "http" {
  security_group_id = aws_security_group.load_balancer.id

  cidr_ipv4   = "0.0.0.0/0"
  from_port   = 80
  ip_protocol = "tcp"
  to_port     = 80
}

resource "aws_vpc_security_group_egress_rule" "all" {
  security_group_id = aws_security_group.load_balancer.id

  cidr_ipv4   = "0.0.0.0/0"
  ip_protocol = "-1"
}

resource "aws_lb" "this" {
  enable_deletion_protection = false
  idle_timeout               = 300
  internal                   = true
  load_balancer_type         = "application"
  preserve_host_header       = false
  subnets                    = var.subnets

  security_groups = concat(
    [aws_security_group.load_balancer.id],
    var.security_groups,
  )
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.this.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type = "fixed-response"

    fixed_response {
      content_type = "text/plain"
      message_body = "404 Not Found"
      status_code  = "404"
    }
  }
}
