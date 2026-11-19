# Example: Data Type - Full

resource "f5xc_data_type" "example" {
  name        = "example-data-type-q7xujfz6"
  namespace   = "default"
  description = "Data type example"

  is_sensitive_data = true
  is_pii            = true
  compliances       = ["GDPR"]

  rules = [{
    pattern_choice = {
      key_value_pattern = {
        key_pattern = {
          type_choice = {
            substring_value = "ssn"
          }
        }
        value_pattern = {
          type_choice = {
            regex_value = "[0-9]{3}-[0-9]{2}-[0-9]{4}"
          }
        }
      }
    }
  }]

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
