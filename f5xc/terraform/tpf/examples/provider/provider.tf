# Copyright (c) 2026 F5, Inc. All rights reserved.

provider "f5xc" {
  url          = "https://url"
  timeout      = "30s"
  rate_limit   = "100"
  api_token    = "abcdsjdsfdshfslkfhsdlafhaslfhsladkfhs"
  api_p12_file = "p12_file.p12"
  limiter = {
    rate  = 30
    burst = 1
  }
}