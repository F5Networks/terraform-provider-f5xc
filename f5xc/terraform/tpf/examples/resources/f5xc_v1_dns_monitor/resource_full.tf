# Example: V1 Dns Monitor - Full

resource "f5xc_v1_dns_monitor" "example" {
  name      = "example-v1-dns-monitor"
  namespace = "system"

  domain      = "example.example.com"
  record_type = "A"
  protocol    = "UDP"

  external_sources = [
    {
      source_choice = {
        aws = {
          regions = ["ca-central-1"]
        }
      }
    }
  ]

  interval_choice = {
    interval_5_mins = true
  }

  on_failure_to_choice = {
    on_failure_to_any = true
  }

  lookup_timeout            = 3000
  source_critical_threshold = 1
  on_failure_count          = 2
  receive                   = "example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  name_servers = [
    {
      name_server = "1.1.1.1"
      port        = 53
    }
  ]
}
