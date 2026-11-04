# Example: Ip Prefix Set - Full

resource "f5xc_ip_prefix_set" "example" {
  name        = "example-ips"
  namespace   = "default"
  description = "Full ip prefix set example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform-team"
  }

  ipv4_prefixes = [
    {
      ipv4_prefix = "10.0.0.0/8"
      description = "Private class A"
    },
    {
      ipv4_prefix = "192.168.0.0/16"
      description = "Private class C"
    }
  ]

  ipv6_prefixes = [
    {
      ipv6_prefix = "2001:0db8:0000:0000:0000:0000:0000:0000/32"
      description = "Documentation prefix"
    }
  ]
}
