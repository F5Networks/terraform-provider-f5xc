# Changelog
All notable customer-facing changes to this project are documented in this file.
## 0.10.6 (November 20, 2026)

### FEATURES
* Add k8s resources to TPF generation and fix uint range validator and root pointer path rendering
* Extend f5xc_api_definition_uploader tftest suite for write-only specs and integration

### IMPROVEMENTS
* Inject build-time provider version via ldflags and propagate to HTTP User-Agent
* Update provider to 2026-10-0 release and align documentation

### BUG FIXES
* Omit reserved ves.io/* system labels on nested labels maps
* Strip backend-injected default NS values on apex ns_record in f5xc_dns_zone
* Fix f5xc_dns_zone RRSet drift, apex NS merging, and rr_set_group lifecycle edge cases
* Fix f5xc_dns_zone RRSet drift, apex NS merging, and rr_set_group lifecycle edge cases
* Generalize ReadStateFallback for static default containers (1a/1b/2a) and convert http_loadbalancer routes to ListNestedAttribute
* Enable ObjectRefType.tenant override for advertise_policy
* Upload SBOMs before provider binaries in publish-release
* Mark aws_vpc_site routing_type_choice as optional and computed
* Enforce ObjectRefType.tenant at plan time for cross-namespace references to shared
* Add fail_if_referred to virtual_site so it can be deleted even if referred in resources excluding virtual_k8s
* Send stored_object description as top-level CreateObjectRequest field
* Handle api_credential, service_credential's API creating kube-config creds in virtual_k8s_namespace instead of namespace
* Encode legacy response_body into response_body_encoded on Read
* Add unit tests for legacy response_body fallback
* Fix historical version pruning in f5xc_api_definition_uploader ref:
* Enforce 5 MiB size limit for string_value in f5xc_api_definition_uploader
* Match f5xc_known_label by (key, value) composite key in Read and ImportState
## 0.10.5 (October 23, 2026)

### BUG FIXES
* Filter backend default rrsets before indexing in apply merge
* Fixes a post-apply Terraform Core consistency error on f5xc_dns_zone
## 0.10.4 (October 16, 2026)

### BUG FIXES
* Regenerate resources with resolveUnknowns guard and targeted bgp leaf healing
## 0.10.3 (October 12, 2026)

### BUG FIXES
* Added import and acceptance test for allowed_domain, mitigated_domain, protected_domain
## 0.10.1 (August 31, 2026)

### IMPROVEMENTS
* Provider produced inconsistent result after apply in user

