resource "f5xc_log_receiver" "example" {
  name      = "example_log_receiver"
  namespace = "example_namespace"
  log_receiver_choice = {
    syslog = {
      syslog = "example_syslog"
    }
  }
  where_choice = {
    site_local = true
  }
}
