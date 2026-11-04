# Example: Geo Location Set - Basic

resource "f5xc_geo_location_set" "example" {
  name        = "example-geo-location-set"
  namespace   = "system"
  description = "Geo location set example"

  location_choice = {
    global = true
  }
}
