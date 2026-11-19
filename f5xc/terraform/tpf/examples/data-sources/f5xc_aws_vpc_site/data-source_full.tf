# Example: Aws Vpc Site - Full

data "f5xc_aws_vpc_site" "example" {
  name      = f5xc_aws_vpc_site.example.name
  namespace = f5xc_aws_vpc_site.example.namespace
}
