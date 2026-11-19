# Example: Known Label Key - Description

resource "f5xc_known_label_key" "example" {
  key         = "example-klk"
  namespace   = "shared"
  description = "Known label key example"
}
