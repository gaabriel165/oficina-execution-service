resource "aws_sqs_queue" "commands_dlq" {
  name                      = "${var.commands_queue_name}-dlq"
  message_retention_seconds = 1209600
  sqs_managed_sse_enabled   = true
}

resource "aws_sqs_queue" "commands" {
  name                       = var.commands_queue_name
  visibility_timeout_seconds = 30
  receive_wait_time_seconds  = 20
  sqs_managed_sse_enabled    = true

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.commands_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })
}

resource "aws_sqs_queue_redrive_allow_policy" "commands_dlq" {
  queue_url = aws_sqs_queue.commands_dlq.id

  redrive_allow_policy = jsonencode({
    redrivePermission = "byQueue"
    sourceQueueArns   = [aws_sqs_queue.commands.arn]
  })
}
