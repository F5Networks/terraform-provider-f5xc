# Example: Malicious User Mitigation - Full

resource "f5xc_malicious_user_mitigation" "example" {
  name        = "example-mum"
  namespace   = "default"
  description = "Full malicious user mitigation example"

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
      },
      {
        threat_level = {
          threat_level_choice = {
            medium = true
          }
        }
        mitigation_action = {
          mitigation_action_choice = {
            captcha_challenge = true
          }
        }
      },
      {
        threat_level = {
          threat_level_choice = {
            high = true
          }
        }
        mitigation_action = {
          mitigation_action_choice = {
            block_temporarily = true
          }
        }
      }
    ]
  }

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
