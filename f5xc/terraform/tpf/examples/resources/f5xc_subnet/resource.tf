resource "f5xc_subnet" "example" {
  name                             = "example_subnet"
  namespace                        = "example_namespace"
  site_dhcp_server_params_internal = [{}]
  site_subnet_params = [{
    site = {
      name      = "example_subnet"
      namespace = "example_namespace"
    }
  }]
}
