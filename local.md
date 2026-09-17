# How to Run `terraform-provider-f5xc` Locally

Follow this step-by-step guide to run and test pre-release builds of the F5 Distributed Cloud (`f5xc`) Terraform
provider downloaded directly from GitHub Releases.

## Prerequisites

* [Terraform CLI](https://developer.hashicorp.com/terraform/install) installed (`>= 1.5.0`).
* A downloaded `.zip` release archive matching your operating system and architecture from
  the [GitHub Releases](https://github.com/F5Networks/terraform-provider-f5xc-preview/releases) page.

## Step 1: Extract the Archive & Detect Version

Extract the downloaded `.zip` file for your platform:

* **macOS (Apple Silicon):** `terraform-provider-f5xc_<VERSION>_darwin_arm64.zip`
* **macOS (Intel):** `terraform-provider-f5xc_<VERSION>_darwin_amd64.zip`
* **Linux:** `terraform-provider-f5xc_<VERSION>_linux_amd64.zip`
* **Windows:** `terraform-provider-f5xc_<VERSION>_windows_amd64.zip`

In your terminal, navigate to the download folder and define the `VERSION` variable:

### macOS / Linux

```bash
# Option 1: Set the version manually
VERSION="0.3.1"

# Option 2: Automatically extract the version from the binary name
VERSION=$(ls terraform-provider-f5xc_v* | sed -E 's/.*_v([0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?).*/\1/' | head -n 1)

echo "Detected Provider Version: ${VERSION}"
```

### Windows (PowerShell)

```powershell
$VERSION = "0.3.1"
```

## Step 2: Set Permissions & Handle macOS Quarantine

### macOS / Linux

Grant execute permissions to the binary:

```bash
chmod +x terraform-provider-f5xc_v*
```

*(macOS Only)* If downloaded via a browser, remove the quarantine attribute:

```bash
xattr -d com.apple.quarantine terraform-provider-f5xc_v*
```

## Step 3: Install the Provider

Choose **Option A** (standard plugin directory mirror) or **Option B** (development override).

### Option A: Local Plugin Directory (Recommended)

Terraform automatically scans your local user plugin directory before querying external registries.

#### 1. Create the plugin directory structure:

* **macOS (Apple Silicon):**
  ```bash
  mkdir -p ~/.terraform.d/plugins/registry.terraform.io/F5Networks/f5xc/${VERSION}/darwin_arm64/
  ```
* **macOS (Intel):**
  ```bash
  mkdir -p ~/.terraform.d/plugins/registry.terraform.io/F5Networks/f5xc/${VERSION}/darwin_amd64/
  ```
* **Linux (AMD64):**
  ```bash
  mkdir -p ~/.terraform.d/plugins/registry.terraform.io/F5Networks/f5xc/${VERSION}/linux_amd64/
  ```
* **Windows (PowerShell):**
  ```powershell
  New-Item -ItemType Directory -Force -Path "$env:APPDATA\terraform.d\plugins\registry.terraform.io\F5Networks\f5xc\$VERSION\windows_amd64"
  ```

#### 2. Move the binary into the directory:

* **macOS (Apple Silicon):**
  ```bash
  mv "terraform-provider-f5xc_v${VERSION}" ~/.terraform.d/plugins/registry.terraform.io/F5Networks/f5xc/${VERSION}/darwin_arm64/
  ```
* **macOS (Intel):**
  ```bash
  mv "terraform-provider-f5xc_v${VERSION}" ~/.terraform.d/plugins/registry.terraform.io/F5Networks/f5xc/${VERSION}/darwin_amd64/
  ```
* **Linux (AMD64):**
  ```bash
  mv "terraform-provider-f5xc_v${VERSION}" ~/.terraform.d/plugins/registry.terraform.io/F5Networks/f5xc/${VERSION}/linux_amd64/
  ```
* **Windows (PowerShell):**
  ```powershell
  Move-Item "terraform-provider-f5xc_v$VERSION.exe" "$env:APPDATA\terraform.d\plugins\registry.terraform.io\F5Networks\f5xc\$VERSION\windows_amd64\"
  ```

#### 3. Configure your Terraform project (`versions.tf`):

Use a flexible version constraint (e.g. `>= 0.1.0` or `~> 0.3`) so any compatible local version is accepted:

```hcl
terraform {
  required_version = ">= 1.5.0"
  required_providers {
    f5xc = {
      source  = "registry.terraform.io/F5Networks/f5xc"
      version = ">= 0.1.0" # Flexible: accepts any local version matching this constraint
    }
  }
}

provider "f5xc" {
  # Add your provider configuration / credentials here
}
```

#### 4. Initialize Terraform:

```bash
terraform init
```

Terraform will detect and use the locally installed provider version without reaching out to the public registry.

### Option B: Development Override via `.terraformrc` (Fastest)

This method bypasses `terraform init` and version checks completely, pointing directly to any directory containing the
binary.

1. Place the extracted binary into a local directory (e.g. `/Users/<username>/bin/` or `C:\terraform-plugins\`).
2. Create or edit your CLI configuration file:
    * **macOS / Linux:** `~/.terraformrc`
    * **Windows:** `%APPDATA%\terraform.rc`
3. Add the following block:
   ```hcl
   provider_installation {
     dev_overrides {
       "F5Networks/f5xc" = "/absolute/path/to/directory/containing/binary"
     }
     direct {}
   }
   ```
4. Run Terraform directly (do **not** run `terraform init`):
   ```bash
   terraform plan
   terraform apply
   ```

*(Note: Terraform will display a `[WARNING] Provider development overrides are in effect...` banner. This is expected
behavior).*

## Verification

To verify that the local provider is active, run:

```bash
terraform version
```

Expected output:

```text
Terraform v1.9.0
on darwin_arm64
+ provider registry.terraform.io/F5Networks/f5xc v0.3.1
```
