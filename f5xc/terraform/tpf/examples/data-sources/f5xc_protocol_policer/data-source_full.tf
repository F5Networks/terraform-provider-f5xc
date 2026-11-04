# Example: Protocol Policer - Full

data "f5xc_protocol_policer" "example" {
  name      = f5xc_protocol_policer.example.name
  namespace = f5xc_protocol_policer.example.namespace
}
