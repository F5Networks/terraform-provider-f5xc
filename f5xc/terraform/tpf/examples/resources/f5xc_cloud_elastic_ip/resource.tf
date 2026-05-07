resource "f5xc_cloud_elastic_ip" "example" {
  namespace = "example_namespace"
  name      = "example_cloud_elastic_ip"
  site_ref = [{
    name      = "example_cloud_elastic_ip"
    namespace = "example_namespace"
  }]
}
