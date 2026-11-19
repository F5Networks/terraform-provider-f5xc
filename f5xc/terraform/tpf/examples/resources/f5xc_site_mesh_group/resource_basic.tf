# Example: Site Mesh Group - Basic

resource "f5xc_site_mesh_group" "example" {
  name      = "example-site-mesh-group"
  namespace = "system"

  mesh_choice = {
    full_mesh = {
      full_mesh_choice = {
        data_plane_mesh = true
      }
    }
  }

  re_fallback_choice = {
    enable_re_fallback = true
  }

  bfd_choice = {
    bfd_disabled = true
  }
}
