output "cluster_arn" {
  value = aws_ecs_cluster.this.arn
}

output "distribution_domain" {
  value = aws_cloudfront_distribution.this.domain_name
}

output "listener_arn" {
  value = aws_lb_listener.http.arn
}
