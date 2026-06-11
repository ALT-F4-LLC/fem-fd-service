#cloud-config
write_files:
  - path: /etc/ecs/ecs.config
    permissions: 0400
    owner: root
    content: |
      ECS_CLUSTER=${cluster_name}
      ECS_ENABLE_SPOT_INSTANCE_DRAINING=true
      ECS_ENABLE_TASK_IAM_ROLE=true
      ECS_LOGLEVEL=debug
