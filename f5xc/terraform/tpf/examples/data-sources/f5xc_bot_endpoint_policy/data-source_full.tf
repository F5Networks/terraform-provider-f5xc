# Example: Bot Endpoint Policy - Full

data "f5xc_bot_endpoint_policy" "example" {
  name      = "example-bot-endpoint-policy"
  namespace = "default"
}
