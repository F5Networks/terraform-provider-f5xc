# Example: Segment Connection - Full

data "f5xc_segment_connection" "example" {
  name      = f5xc_segment_connection.example.name
  namespace = f5xc_segment_connection.example.namespace
}
