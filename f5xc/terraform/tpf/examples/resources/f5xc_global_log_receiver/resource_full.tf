# Example: Global Log Receiver - Full

resource "f5xc_global_log_receiver" "example" {
  name        = "example-global-log-receiver"
  namespace   = "system"
  description = "Full global log receiver example"

  filter_choice = {
    ns_list = {
      namespaces = ["default"]
    }
  }

  log_type_choice = {
    request_logs = {
      sampling_choice = {
        sampled = true
      }
    }
  }

  receiver_choice = {
    http_receiver = {
      auth_choice = {
        auth_none = true
      }
      tls_choice = {
        no_tls = true
      }
      uri = "http://example.com:9000/logs"
    }
  }

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }
}
