# Example: Geo Location Set - Countries

resource "f5xc_geo_location_set" "example" {
  name        = "example-geo-location-set-countries"
  namespace   = "system"
  description = "Geo location set example with countries"

  location_choice = {
    custom_geo_location_selector = {
      expressions = ["country in (US, DE, FR)"]
    }
  }
}
