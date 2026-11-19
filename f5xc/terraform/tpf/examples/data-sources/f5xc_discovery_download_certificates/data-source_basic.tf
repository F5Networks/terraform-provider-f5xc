# Example: Discovery Download Certificates - Full

data "f5xc_discovery_download_certificates" "example" {
  name      = "example-discovery"
  namespace = "default"
}
