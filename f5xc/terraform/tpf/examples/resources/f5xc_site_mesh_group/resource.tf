resource "f5xc_site_mesh_group" "example" {
  name      = "example_site_mesh_group"
  namespace = "example_namespace"
  bfd_choice = {
    bfd_enabled = {
      bfd_enabled = "example_bfd_enabled"
    }
  }
  re_fallback = {
    enable_re_fallback = true
  }
}
