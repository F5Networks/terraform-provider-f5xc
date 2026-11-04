# Example: Cloud Elastic Ip - Full

data "f5xc_cloud_elastic_ip" "example" {
  name      = f5xc_cloud_elastic_ip.example.name
  namespace = f5xc_cloud_elastic_ip.example.namespace
}
