resource "f5xc_app_firewall" "example" {
  name      = "example_app_firewall"
  namespace = "example_namespace"
  allowed_response_codes_choice = {
    allow_all_response_codes = true
  }
  anonymization_setting = {
    disable_anonymization = true
  }
  blocking_page_choice = {
    use_default_blocking_page = true
  }
  enhance_with_ai_choice = {
    disable_ai_enhancements = true
  }
  enforcement_mode_choice = {
    blocking = true
  }
  detection_setting_choice = {
    default_detection_settings = true
  }
}
