# Example: Known Label - Full

data "f5xc_known_label" "example" {
  key       = f5xc_known_label.example.key
  namespace = f5xc_known_label.example.namespace
}
