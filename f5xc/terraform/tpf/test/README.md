# Acceptance Testing Quickstart

## Prerequisites

- Go installed and available on PATH.
- Access to an F5XC tenant and API credentials (not required for REPLAYING mode).

## One-Time Environment Setup

Set required environment variables in your shell profile or current terminal session.

- Required:
  - `F5XCProvider_URL`
- Required authentication (choose one):
  - `F5XCProvider_API_TOKEN`
  - or `F5XCProvider_API_KEY` and `F5XCProvider_API_CERT`
  - or `F5XCProvider_API_P12_FILE` and `VES_P12_PASSWORD`
- Optional:
  - `F5XCProvider_NAMESPACE` (for all testing, set this to `terraform-acceptance-test`)
  - `F5XCProvider_TIMEOUT`

### Resource-specific environment variables

Some acceptance tests need additional inputs beyond the shared provider
credentials.

- `f5xc_cloud_link`
  - `F5XC_CLOUD_LINK_AWS_CRED_NAME`
  - Optional override for the existing AWS cloud credential referenced by the
    `f5xc_cloud_link` acceptance tests.
  - The test uses a pre-existing AWS Direct Connect profile and does not create
    a cloud credential resource as part of the testcase.
  - When `F5XCProvider_URL` targets `customer1.crt.volterra.us`, the test falls
    back to the QA-aligned credential `orch-test-user-site-views-aws-vpc-dc-new`
    if this variable is unset.

- `f5xc_infraprotect_asn_prefix`
  - `F5XC_INFRAPROTECT_APPROVED_ASN_NAME`
  - Optional override for the ASN reference used by
    `f5xc_infraprotect_asn_prefix` acceptance tests.
  - When `F5XC_INFRAPROTECT_APPROVED_ASN_NAME` is set, tests use that
    pre-approved ASN and skip the ASN create/approve pre-step.
  - When unset, tests keep the default flow: create `f5xc_infraprotect_asn`
    and call `UpdateASNReviewStatus` before creating the ASN prefix.

Example:

```sh
export F5XC_CLOUD_ELASTIC_IP_SITE_NAME=ak-ce5
export F5XC_CLOUD_LINK_AWS_CRED_NAME=orch-test-user-site-views-aws-vpc-dc-new
export F5XC_INFRAPROTECT_APPROVED_ASN_NAME=<approved-asn-name>
export F5XC_INFRAPROTECT_APPROVED_ASN_NAMESPACE=system
```

### Required test namespace

Use the same namespace for all tests (LIVE, RECORDING, REPLAYING, and import validation):

```sh
export F5XCProvider_NAMESPACE=terraform-acceptance-test
```

### Validate environment quickly:
```sh
make acc-env-check
```

### List all acceptance targets:
```sh
make acc-help
```

---

## VCR Acceptance Testing

All acceptance tests are VCR-aware.  Three execution modes are available via
the `VCR_MODE` environment variable:

| `VCR_MODE`  | Behaviour |
|-------------|-----------|
| *(unset)*   | **LIVE** – direct API calls, no cassettes |
| `RECORDING` | Live API calls + write cassette files to disk |
| `REPLAYING` | Read-only replay from cassettes, no live API calls |

---

## Cassette Layout

Cassettes are stored alongside the test files that generate them:

```
test/acctest/
  resource/
    cassettes/          ← resource test cassettes
      TestAccResourceAutoF5XCOriginPool_basic.yaml
      ...
    resource_auto_f5xc_namespace_test.go
    ...
  data/
    cassettes/          ← data source test cassettes
      TestAccDataSourceAutoF5XCFoo_basic.yaml
      ...
```

Each test writes to a single cassette file named after `t.Name()` (slashes and
spaces replaced with underscores).  All HTTP interactions for a single test run
are stored in one file.

---

## Make Targets

### Resource tests

Record one resource test by name:
```sh
make acc-vcr-record-one TEST=TestAccResourceAutoF5XCOriginPool_basic
```

Replay one resource test by name:
```sh
make acc-vcr-replay-one TEST=TestAccResourceAutoF5XCOriginPool_basic
```

Remove all resource cassettes:
```sh
make acc-vcr-clean
```

Run all resource acceptance tests against the live API:
```sh
make acc-live
```

Record cassettes for all resource tests:
```sh
make acc-vcr-record
```

Replay all resource tests from cassettes (no credentials needed):
```sh
make acc-vcr-replay
```

### Data source tests

The same targets are available with the `acc-ds-` prefix:

```sh
make acc-ds-live
make acc-ds-vcr-record
make acc-ds-vcr-replay
make acc-ds-vcr-record-one TEST=TestAccDataSourceAutoF5XCFoo_basic
make acc-ds-vcr-replay-one TEST=TestAccDataSourceAutoF5XCFoo_basic
make acc-ds-vcr-clean
```

### Combined targets

```sh
make acc-all-live         # run all acceptance tests live
make acc-all-vcr-record   # record all cassettes
make acc-all-vcr-replay   # replay all cassettes
make acc-all-vcr-clean    # remove all cassettes
```

---
## Managing Cassettes

### Recording a cassette

```sh
# Set your live credentials:
export F5XCProvider_URL=https://your-tenant.console.ves.volterra.io/api
export F5XCProvider_API_TOKEN=<your-token>

make acc-vcr-record-one TEST=TestAccResourceAutoF5XCOriginPool_basic
```

The cassette is written to
`test/acctest/resource/cassettes/TestAccResourceAutoF5XCOriginPool_basic.yaml`.

### Replaying a cassette

No credentials are required:

```sh
export F5XCProvider_URL=https://your-tenant.console.ves.volterra.io/api

make acc-vcr-replay-one TEST=TestAccResourceAutoF5XCOriginPool_basic
```

### Refreshing a stale cassette

If the API response schema changes or a cassette drifts from the test, delete
the cassette file and re-record:

```sh
rm test/acctest/resource/cassettes/TestAccResourceAutoF5XCOriginPool_basic.yaml
make acc-vcr-record-one TEST=TestAccResourceAutoF5XCOriginPool_basic
```

### Sensitive data

The VCR framework automatically redacts the following request headers before
saving cassettes to disk:

- `Authorization`
- `X-Volterra-Apigw-Tenant`
- `Cookie`

Do **not** commit raw cassettes that contain unredacted credentials.

---

## Recommended First Run

```sh
make acc-env-check
make acc-vcr-record-one TEST=TestAccResourceAutoF5XCOriginPool_basic
make acc-vcr-replay-one TEST=TestAccResourceAutoF5XCOriginPool_basic
```

This validates live access and confirms deterministic replay on your machine.

---

## Blindfold Secret Encryption in Tests

### What is Blindfold?

Blindfold is F5 XC's secret encryption mechanism. A blindfold-encrypted blob is encrypted with the **public key of a specific tenant/customer**. The server can only decrypt it using the corresponding private key.

### Why Blindfold in Tests?

Some resources (e.g., `api_crawler`, `certificate`) support blindfold-encrypted secrets. Tests that exercise this feature must use tenant-specific encrypted blobs. A blob encrypted for one tenant cannot be decrypted by another tenant's server.

### How It Works

1. **Encrypted Blobs per Tenant:** Each tenant has a unique blindfold encryption key pair. Test blobs are pre-encrypted for specific tenants and stored as base64 strings in `const.go`:
   - `testBlindFoldKeyB64Staging` — for `playground.staging.volterra.us`
   - `testBlindFoldKeyB64Crt` — for `customer1.crt.volterra.us`

2. **Runtime Tenant Detection:** The helper function `blindFoldKeyB64(t testing.TB)` in `helpers.go` reads the `F5XCProvider_URL` environment variable and returns the correct blob for the target tenant:
   ```go
   func blindFoldKeyB64(t testing.TB) string {
       u := os.Getenv("F5XCProvider_URL")
       switch {
       case strings.Contains(u, "playground.staging.volterra.us"):
           return testBlindFoldKeyB64Staging
       case strings.Contains(u, "customer1.crt.volterra.us"):
           return testBlindFoldKeyB64Crt
       default:
           t.Fatalf("blindFoldKeyB64: no blindfold key configured for URL %q", u)
       }
   }
   ```

3. **Test Usage:** Tests call `blindFoldKeyB64(t)` to get the correct blob for the current tenant:
   ```go
   Config: testAccResourceAutoF5XCApiCrawlerConfig_withBlindfoldSecret(
       name, namespace, "crawluser", "string:///"+blindFoldKeyB64(t),
   ),
   ```

### Adding Support for a New Tenant

To add a new tenant:

1. **Generate the encrypted blob** for the new tenant (contact F5 XC team or use tenant's encryption tools).
2. **Add the constant** to `const.go`:
   ```go
   const testBlindFoldKeyB64NewTenant = "base64-encoded-blob-here"
   ```
3. **Add a case** to `blindFoldKeyB64()` in `helpers.go`:
   ```go
   case strings.Contains(u, "newtenant.volterra.us"):
       return testBlindFoldKeyB64NewTenant
   ```

---

## Unit Testing

Unit tests validate resource schema definitions.

- Attributes are defined with correct types (string, int64, list, map, object, etc.)
- Required/optional/computed flags are properly set
- Validators are registered where expected (e.g., string length, enum values)
- Nested objects and attributes have correct structure

### Running unit tests

```sh
make test-schema
```

---

## CI/CD Webhook Execution (`.tftest.hcl` Suites from `omnigen-data`)

In addition to Go acceptance tests (`acceptance-test`), `.gitlab-ci.yml` provides the `tftest` job to execute the native Terraform 1.6+ `.tftest.hcl` test suites stored in `omnigen-data` against a compiled provider binary using `dev_overrides`.

Webhook and API pipeline triggers use the `TEST_TYPE` variable to select which test suite runs (while skipping `lint`, `unittest`, `build`, `container`, `release`, `publish-release`, and security scans):

| Variable | Default | Description |
|---|---|---|
| `TEST_TYPE` | `acc` | Selects the test job for `trigger`/`api` pipelines: `acc` (`acceptance-test`), `tftest` (`tftest`), or `all` (both sequentially). |
| `OMNIGEN_DATA_BRANCH` | `main` | Branch of `omnigen-data` to clone. |
| `OMNIGEN_DATA_RELEASE_DIR` | *(latest `release-*`)* | Optional release subdirectory inside `omnigen-data` (e.g. `release-2026-10-0`). |
| `TFTEST_RESOURCES` | *(all)* | Optional comma-separated list of resources to execute (e.g. `dns_zone,origin_pool` or `f5xc_dns_zone`). |
| `TFTEST_TIMEOUT` | `300s` | Per-suite execution timeout passed to `timeout`. |
| `TFTEST_EXCLUDE_BEFORE` | `true` | Exclude `*_before.tftest.hcl` baseline regression-proof files (which assert failure on uncompensated providers). Set to `false` to include them. |
| `TERRAFORM_VERSION` | *(latest / `1.14.8`)* | Optional Terraform CLI version override. |

### Triggering via GitLab Webhook (`curl`)

Run all `.tftest.hcl` suites against the default `omnigen-data` (`main`) branch and latest `release-*` directory:

```sh
curl --request POST \
  --form token="<TRIGGER_TOKEN>" \
  --form ref="release-2026-10-0" \
  --form "variables[TEST_TYPE]=tftest" \
  "https://gitlab.com/api/v4/projects/50750365/trigger/pipeline"
```

Run specific resources or override `omnigen-data` branch, release directory, timeout, or Terraform version:

```sh
curl --request POST \
  --form token="<TRIGGER_TOKEN>" \
  --form ref="<branch-or-tag>" \
  --form "variables[TEST_TYPE]=tftest" \
  --form "variables[TFTEST_RESOURCES]=f5xc_dns_zone,origin_pool" \
  --form "variables[OMNIGEN_DATA_BRANCH]=main" \
  --form "variables[OMNIGEN_DATA_RELEASE_DIR]=release-2026-10-0" \
  --form "variables[TFTEST_TIMEOUT]=300s" \
  --form "variables[TFTEST_EXCLUDE_BEFORE]=true" \
  --form "variables[TERRAFORM_VERSION]=1.14.8" \
  "https://gitlab.com/api/v4/projects/50750365/trigger/pipeline"
```

### Running `.tftest.hcl` Suites Locally

```sh
# 1. Build the provider binary into a dedicated directory
mkdir -p .tftest_bin
CGO_ENABLED=0 go build -mod=vendor -ldflags="-s -w" \
  -o .tftest_bin/terraform-provider-f5xc ./cmd/terraform-provider-f5xc

# 2. Export credentials and target omnigen-data test directory
export F5XCProvider_URL="https://customer1.crt.volterra.us/api"
export F5XCProvider_API_TOKEN="<your-token>"
export PROVIDER_BIN_DIR="${PWD}/.tftest_bin"
export TFTEST_DIR="/path/to/omnigen-data/release-2026-10-0/tests"
export TFTEST_RESOURCES="f5xc_dns_zone"

# 3. Execute the batch runner
./scripts/run-tftest.sh
```
