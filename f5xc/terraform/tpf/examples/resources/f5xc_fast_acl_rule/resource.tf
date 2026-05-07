resource "f5xc_fast_acl_rule" "example" {
  name      = "example_fast_acl_rule"
  namespace = "example_namespace"
  source = {
    prefix = {
      prefix = "example_prefix"
    }
  }
  action = {}
  port   = [{}]
}
