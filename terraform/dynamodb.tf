locals {
  tables = toset(["work-orders", "catalog", "outbox", "processed-messages"])
}

resource "aws_dynamodb_table" "this" {
  for_each = local.tables

  name         = "${var.table_prefix}-${each.value}"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  server_side_encryption {
    enabled = true
  }

  point_in_time_recovery {
    enabled = var.point_in_time_recovery
  }
}
