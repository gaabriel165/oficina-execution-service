variable "region" {
  description = "AWS region to provision into"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Name of the project shared by every repository"
  type        = string
  default     = "oficina-api"
}

variable "service_name" {
  description = "Name prefix applied to the resources of this microservice"
  type        = string
  default     = "oficina-execution-service"
}

variable "state_bucket" {
  description = "S3 bucket holding the Terraform state of the other repositories"
  type        = string
  default     = "oficina-api-terraform-state-728750563430"
}

variable "kubernetes_namespace" {
  description = "Namespace of the workload, trusted by the pod role"
  type        = string
  default     = "oficina-execution"
}

variable "commands_queue_name" {
  description = "Queue with the saga commands consumed by this microservice"
  type        = string
  default     = "oficina-execution-commands"
}

variable "saga_replies_queue_name" {
  description = "Queue owned by the service order microservice that receives the saga replies"
  type        = string
  default     = "oficina-saga-replies"
}

variable "max_receive_count" {
  description = "Deliveries of a command before it is moved to the dead-letter queue"
  type        = number
  default     = 5
}

variable "table_prefix" {
  description = "Prefix of the DynamoDB tables, matching DYNAMODB_TABLE_PREFIX of the application"
  type        = string
  default     = "oficina-execution"
}

variable "point_in_time_recovery" {
  description = "Continuous backups of the DynamoDB tables"
  type        = bool
  default     = false
}

variable "alert_email" {
  description = "E-mail subscribed to the alerts topic. Leave empty to create the topic without subscription"
  type        = string
  default     = ""
}
