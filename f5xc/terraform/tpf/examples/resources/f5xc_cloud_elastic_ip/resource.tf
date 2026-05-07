resource "f5xc_cloud_elastic_ip" "example" {
  name      = "example_cloud_elastic_ip"
  namespace = "example_namespace"
  site_ref = [{
    name      = "example_cloud_elastic_ip"
    namespace = "example_namespace"
  }]
}
