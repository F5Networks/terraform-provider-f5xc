# Example: Global Log Receiver - Full

data "f5xc_global_log_receiver" "example" {
  name      = f5xc_global_log_receiver.example.name
  namespace = f5xc_global_log_receiver.example.namespace
}
