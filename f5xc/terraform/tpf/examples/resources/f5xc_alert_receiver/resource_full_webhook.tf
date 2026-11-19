# Example: Alert Receiver - Full Webhook

resource "f5xc_alert_receiver" "example" {
  name        = "example-alert-receiver"
  namespace   = "default"
  description = "Webhook receiver example updated"

  labels = {
    env  = "example"
    team = "sre"
  }

  annotations = {
    owner = "platform"
  }
  receiver_choice = {
    webhook = {
      url = {
        secret_info_choice = {
          clear_secret_info = {
            url = "string:///aHR0cHM6Ly9leGFtcGxlLmNvbS93ZWJob29r"
          }
        }
      }
      http_config = {
        auth_choice = {
          basic_auth = {
            user_name = "api-user"
            password = {
              secret_info_choice = {
                clear_secret_info = {
                  url = "string:///ZXhhbXBsZS1wYXNzd29yZA=="
                }
              }
            }
          }
        }
        enable_http2     = true
        follow_redirects = true
        tls_choice = {
          use_tls = {
            sni_choice = {
              sni = "alerts.example.com"
            }
            min_version = ["TLSv1_2"]
            max_version = ["TLSv1_3"]
            server_validation_choice = {
              volterra_trusted_ca = true
            }
          }
        }
      }
    }
  }
}
