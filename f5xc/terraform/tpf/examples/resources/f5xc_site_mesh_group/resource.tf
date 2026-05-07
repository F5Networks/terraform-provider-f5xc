resource "f5xc_site_mesh_group" "example" {
  name      = "example_site_mesh_group"
  namespace = "example_namespace"
  re_fallback = {
    enable_re_fallback = true
  }
  bfd_choice = {
    bfd_disabled = true
  }
}
