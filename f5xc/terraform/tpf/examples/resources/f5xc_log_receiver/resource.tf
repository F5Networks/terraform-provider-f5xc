resource "f5xc_log_receiver" "example" {
  namespace = "example_namespace"
  name      = "example_log_receiver"
  log_receiver_choice = {
    syslog = {
      syslog = "example_syslog"
    }
  }
  where_choice = {
    site_local = true
  }
}
