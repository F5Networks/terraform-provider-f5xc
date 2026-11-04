# Example: Log Receiver - Full

resource "f5xc_log_receiver" "example" {
  name        = "example-log-receiver"
  namespace   = "default"
  description = "Full log receiver example"

  labels = {
    env  = "example"
    tier = "shared"
  }

  annotations = {
    owner = "platform-team"
  }

  where_choice = {
    site_local = true
  }

  log_receiver_choice = {
    syslog = {
      mode_choice = {
        udp_server = {
          server_name = "full-a.example.com"
          port        = 6514
        }
      }
      format_choice = {
        syslog_rfc5424 = 10485760
      }
    }
  }
}
