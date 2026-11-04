# Example: Segment Connection - Full

resource "f5xc_segment" "source" {
  name      = "example-source-segment"
  namespace = "system"
}

resource "f5xc_segment" "destination" {
  name      = "example-destination-segment"
  namespace = "system"
}

resource "f5xc_segment_connection" "example" {
  name        = "default-segment-connection"
  namespace   = "system"
  description = "segment connection full"

  labels = {
    env  = "example"
    team = "platform"
  }

  annotations = {
    owner = "example-owner"
  }

  connections = [
    {
      connection_choice = {
        direct = true
      }
      source_segments = [
        {
          name      = f5xc_segment.source.name
          namespace = f5xc_segment.source.namespace
        }
      ]
      destination_segments = [
        {
          name      = f5xc_segment.destination.name
          namespace = f5xc_segment.destination.namespace
        }
      ]
    }
  ]
}
