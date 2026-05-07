resource "f5xc_fast_acl_rule" "example" {
  name      = "example_fast_acl_rule"
  namespace = "example_namespace"
  port      = [{}]
  source = {
    prefix = {
      prefix = "example_prefix"
    }
  }
  action = {}
}
