locals {
  saga_replies_queue_arn = "arn:aws:sqs:${var.region}:${data.aws_caller_identity.current.account_id}:${var.saga_replies_queue_name}"
}

data "aws_iam_policy_document" "pod_assume_role" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [local.oidc_provider_arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider}:sub"
      values   = ["system:serviceaccount:${var.kubernetes_namespace}:${var.service_name}"]
    }

    condition {
      test     = "StringEquals"
      variable = "${local.oidc_provider}:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "pod" {
  statement {
    sid = "ConsumeCommands"
    actions = [
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:ChangeMessageVisibility",
      "sqs:GetQueueAttributes",
      "sqs:GetQueueUrl",
    ]
    resources = [aws_sqs_queue.commands.arn]
  }

  statement {
    sid       = "PublishSagaReplies"
    actions   = ["sqs:SendMessage", "sqs:GetQueueUrl"]
    resources = [local.saga_replies_queue_arn]
  }

  statement {
    sid = "ReadAndWriteTables"
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:UpdateItem",
      "dynamodb:DeleteItem",
      "dynamodb:Query",
      "dynamodb:Scan",
      "dynamodb:DescribeTable",
      "dynamodb:TransactWriteItems",
      "dynamodb:TransactGetItems",
      "dynamodb:BatchGetItem",
      "dynamodb:BatchWriteItem",
    ]
    resources = [for table in aws_dynamodb_table.this : table.arn]
  }
}

resource "aws_iam_role" "pod" {
  name               = "${var.service_name}-pod"
  description        = "Assumed by the pods of this microservice through the cluster OIDC provider (IRSA)"
  assume_role_policy = data.aws_iam_policy_document.pod_assume_role.json
}

resource "aws_iam_role_policy" "pod" {
  name   = "${var.service_name}-pod"
  role   = aws_iam_role.pod.id
  policy = data.aws_iam_policy_document.pod.json
}
