# Changelog
All notable customer-facing changes to this project are documented in this file.
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

