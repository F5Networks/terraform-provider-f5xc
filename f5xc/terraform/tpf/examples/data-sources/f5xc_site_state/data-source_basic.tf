# Basic Example: Query CE site state
data "f5xc_site_state" "example" {
  name      = "my-ce-site"
  namespace = "system"
}

output "ce_site_state" {
  value = data.f5xc_site_state.example.site_state
}

output "ce_is_online" {
  value = data.f5xc_site_state.example.is_online
}
