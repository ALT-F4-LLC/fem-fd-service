output "execution_role_arn" {
  value = aws_iam_role.execution.arn
}

output "execution_role_name" {
  value = aws_iam_role.execution.name
}

output "task_role_arn" {
  value = aws_iam_role.task.arn
}

output "task_role_name" {
  value = aws_iam_role.task.name
}

output "service_role_arn" {
  value = aws_iam_role.service.arn
}

output "service_role_name" {
  value = aws_iam_role.service.name
}
