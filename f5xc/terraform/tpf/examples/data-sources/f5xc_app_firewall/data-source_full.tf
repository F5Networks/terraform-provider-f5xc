# Example: App Firewall - Full

data "f5xc_app_firewall" "example" {
  name      = f5xc_app_firewall.example.name
  namespace = f5xc_app_firewall.example.namespace
}
