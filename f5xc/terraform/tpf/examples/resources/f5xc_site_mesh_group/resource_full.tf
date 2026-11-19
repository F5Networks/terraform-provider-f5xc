# Example: Site Mesh Group - Full

resource "f5xc_site_mesh_group" "example" {
  name        = "example-site-mesh-group"
  namespace   = "system"
  description = "Site mesh group full example"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "mesh-owner"
  }

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
    bfd_enabled = {
      transmit_interval_milliseconds = 1200
      receive_interval_milliseconds  = 1200
      multiplier                     = 4
    }
  }
}
