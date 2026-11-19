# Example: Public IP - Full

data "f5xc_public_ip" "example" {
  name      = "public-ipv4"
  namespace = "system"
}
