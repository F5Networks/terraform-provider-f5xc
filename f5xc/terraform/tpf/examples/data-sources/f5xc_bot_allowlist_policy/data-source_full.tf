# Example: Bot Allowlist Policy - Full

data "f5xc_bot_allowlist_policy" "example" {
  name      = "example-bot-allowlist-policy"
  namespace = "default"
}
