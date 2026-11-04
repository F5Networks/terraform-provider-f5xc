# Example: Malicious User Mitigation - Basic

resource "f5xc_malicious_user_mitigation" "example" {
  name        = "example-mum"
  namespace   = "default"
  description = "Basic malicious user mitigation example"

  mitigation_type = {
    rules = [
      {
        threat_level = {
          threat_level_choice = {
            low = true
          }
        }
        mitigation_action = {
          mitigation_action_choice = {
            javascript_challenge = true
          }
        }
      }
    ]
  }
}
