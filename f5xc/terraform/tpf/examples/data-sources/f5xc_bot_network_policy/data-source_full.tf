# Example: Bot Network Policy - Full

data "f5xc_bot_network_policy" "example" {
  name      = "example-bot-network-policy"
  namespace = "default"
}
