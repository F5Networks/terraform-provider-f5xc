//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveMetadata_UsesGeneratedResourceLinksOverride(t *testing.T) {
	cfg := &Config{
		DefaultCategory: "General",
		Resources: map[string]DocMetadata{
			"origin_pool": {
				Subcategory: "App Security & Load Balancing",
				APIURL:      "https://docs.cloud.f5.com/docs-v2/api/config-url",
			},
		},
		DataSources:    map[string]DocMetadata{},
		PrefixMappings: map[string]string{},
	}

	links := &LinksConfig{Resources: map[string]string{"origin_pool": "https://docs.cloud.f5.com/docs-v2/api/generated-url"}}

	meta := resolveMetadata("origin_pool", "resources", cfg, links)
	if meta.APIURL != "https://docs.cloud.f5.com/docs-v2/api/generated-url" {
		t.Fatalf("expected generated resource links override URL, got %q", meta.APIURL)
	}
	if meta.Subcategory != "App Security & Load Balancing" {
		t.Fatalf("expected subcategory to come from categories config, got %q", meta.Subcategory)
	}
}

func TestMergeLinksConfig_UsesOverrideOnlyForExplicitEntries(t *testing.T) {
	base := &LinksConfig{
		Resources: map[string]string{
			"api_crawler":      "https://docs.cloud.f5.com/docs-v2/api/api-crawler",
			"allowed_domain":   "https://docs.cloud.f5.com/docs-v2/api/allowed-domain",
			"protected_domain": "https://docs.cloud.f5.com/docs-v2/api/protected-domain",
		},
		DataSources: map[string]string{
			"active_alert_policies": "https://docs.cloud.f5.com/docs-v2/api/namespace",
		},
	}
	override := &LinksConfig{
		Resources: map[string]string{
			"api_crawler": "https://docs.cloud.f5.com/docs-v2/api/api-sec-api-crawler",
		},
	}

	merged := mergeLinksConfig(base, override)

	if merged.Resources["api_crawler"] != "https://docs.cloud.f5.com/docs-v2/api/api-sec-api-crawler" {
		t.Fatalf("expected override resource URL to win, got %q", merged.Resources["api_crawler"])
	}
	if merged.Resources["allowed_domain"] != "https://docs.cloud.f5.com/docs-v2/api/allowed-domain" {
		t.Fatalf("expected non-overridden resource URL to remain from generated data, got %q", merged.Resources["allowed_domain"])
	}
	if merged.DataSources["active_alert_policies"] != "https://docs.cloud.f5.com/docs-v2/api/namespace" {
		t.Fatalf("expected generated data source URL to remain unchanged, got %q", merged.DataSources["active_alert_policies"])
	}
}

func TestResolveMetadata_UsesGeneratedDataSourceLinksOverride(t *testing.T) {
	cfg := &Config{
		DefaultCategory: "General",
		Resources:       map[string]DocMetadata{},
		DataSources: map[string]DocMetadata{
			"active_alert_policies": {
				Subcategory: "General",
				APIURL:      "https://docs.cloud.f5.com/docs-v2/api/old-url",
			},
		},
		PrefixMappings: map[string]string{},
	}

	links := &LinksConfig{DataSources: map[string]string{"active_alert_policies": "https://docs.cloud.f5.com/docs-v2/api/new-url"}}

	meta := resolveMetadata("active_alert_policies", "data-sources", cfg, links)
	if meta.APIURL != "https://docs.cloud.f5.com/docs-v2/api/new-url" {
		t.Fatalf("expected data-source generated links override URL, got %q", meta.APIURL)
	}
	if meta.Subcategory != "General" {
		t.Fatalf("expected subcategory to remain from categories config, got %q", meta.Subcategory)
	}
}

func TestLoadConfig_LoadsCategoriesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "categories.yaml")
	content := strings.Join([]string{
		"default_category: General",
		"prefix_mappings:",
		"  api_: API Security & Management",
		"resources:",
		"  origin_pool:",
		"    subcategory: App Security & Load Balancing",
		"",
	}, "\n")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed writing categories config: %v", err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("expected loadConfig to parse categories config: %v", err)
	}
	if cfg.DefaultCategory != "General" {
		t.Fatalf("default category = %q, want %q", cfg.DefaultCategory, "General")
	}
	if cfg.PrefixMappings["api_"] != "API Security & Management" {
		t.Fatalf("prefix mapping = %q, want %q", cfg.PrefixMappings["api_"], "API Security & Management")
	}
	if cfg.Resources["origin_pool"].Subcategory != "App Security & Load Balancing" {
		t.Fatalf("resource subcategory = %q, want %q", cfg.Resources["origin_pool"].Subcategory, "App Security & Load Balancing")
	}
}

func TestLoadConfig_LoadsStringStyleCategoriesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "categories.yaml")
	content := strings.Join([]string{
		"default_category: General",
		"prefix_mappings:",
		"  api_: API Security & Management",
		"resources:",
		"  origin_pool: App Security & Load Balancing",
		"  http_loadbalancer: App Security & Load Balancing",
		"",
	}, "\n")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed writing categories config: %v", err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("expected loadConfig to parse string-style categories config: %v", err)
	}
	if cfg.Resources["origin_pool"].Subcategory != "App Security & Load Balancing" {
		t.Fatalf("resource subcategory = %q, want %q", cfg.Resources["origin_pool"].Subcategory, "App Security & Load Balancing")
	}
	if cfg.Resources["http_loadbalancer"].Subcategory != "App Security & Load Balancing" {
		t.Fatalf("resource subcategory = %q, want %q", cfg.Resources["http_loadbalancer"].Subcategory, "App Security & Load Balancing")
	}
}

func TestLoadLinksConfig_MissingFileReturnsEmpty(t *testing.T) {
	links, err := loadLinksConfig(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("expected missing links file to be non-fatal, got error: %v", err)
	}
	if len(links.Resources) != 0 || len(links.DataSources) != 0 {
		t.Fatalf("expected empty links maps for missing file")
	}
}

func TestUpsertAPIReference_InsertsAndIsIdempotent(t *testing.T) {
	body := "\n# f5xc_origin_pool (Resource)\n\nOrigin Pool resource description.\n\n## Example Usage\n"
	apiURL := "https://docs.cloud.f5.com/docs-v2/api/views-origin-pool"

	first := upsertAPIReference(body, apiURL)
	if !strings.Contains(first, "- [API documentation]("+apiURL+")") {
		t.Fatalf("expected API documentation line to be inserted")
	}

	second := upsertAPIReference(first, apiURL)
	if first != second {
		t.Fatalf("expected idempotent API Reference insertion/update")
	}
}

func TestUpdateDocFile_UpdatesSubcategoryAndAPIReference(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "origin_pool.md")
	original := "---\npage_title: \"f5xc_origin_pool Resource\"\ndescription: |-\n  origin pool description\n---\n\n# f5xc_origin_pool (Resource)\n\nOrigin Pool docs description.\n\n## Schema\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatalf("failed writing test markdown: %v", err)
	}

	meta := DocMetadata{
		Subcategory: "App Security & Load Balancing",
		APIURL:      "https://docs.cloud.f5.com/docs-v2/api/views-origin-pool",
	}

	updated, err := updateDocFile(path, meta, true)
	if err != nil {
		t.Fatalf("updateDocFile returned error: %v", err)
	}
	if !updated {
		t.Fatalf("expected updateDocFile to report file changed")
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading updated markdown: %v", err)
	}
	content := string(out)
	if !strings.Contains(content, "subcategory: \"App Security & Load Balancing\"") {
		t.Fatalf("expected subcategory to be set in frontmatter")
	}
	if !strings.Contains(content, "- [API documentation](https://docs.cloud.f5.com/docs-v2/api/views-origin-pool)") {
		t.Fatalf("expected API reference line to be inserted")
	}

	updatedAgain, err := updateDocFile(path, meta, true)
	if err != nil {
		t.Fatalf("updateDocFile second pass error: %v", err)
	}
	if updatedAgain {
		t.Fatalf("expected second pass to be idempotent and report no changes")
	}
}

func TestResolveMetadata_AttachmentResourceInheritsSubcategoryAndParentLink(t *testing.T) {
	cfg := &Config{
		DefaultCategory: "General",
		Resources:       map[string]DocMetadata{},
		DataSources:     map[string]DocMetadata{},
		PrefixMappings: map[string]string{
			"securemesh_": "Site Management",
		},
	}

	links := &LinksConfig{
		Resources: map[string]string{
			"tcp_loadbalancer":   "https://docs.cloud.f5.com/docs-v2/api/views-tcp-loadbalancer",
			"securemesh_site":    "https://docs.cloud.f5.com/docs-v2/api/views-securemesh-site",
			"securemesh_site_v2": "https://docs.cloud.f5.com/docs-v2/api/views-securemesh-site-v2",
		},
	}

	tcpMeta := resolveMetadata("tcp_loadbalancer_service_policy_attachments", "resources", cfg, links)
	if tcpMeta.Subcategory != "App Security & Load Balancing" {
		t.Fatalf("expected attachment subcategory %q, got %q", "App Security & Load Balancing", tcpMeta.Subcategory)
	}
	if tcpMeta.APIURL != "https://docs.cloud.f5.com/docs-v2/api/views-tcp-loadbalancer" {
		t.Fatalf("expected parent TCP loadbalancer API URL, got %q", tcpMeta.APIURL)
	}

	smMeta := resolveMetadata("securemesh_site_v2_forward_proxy_policy_attachments", "resources", cfg, links)
	if smMeta.Subcategory != "App Security & Load Balancing" {
		t.Fatalf("expected attachment subcategory %q even when prefix_mappings match, got %q", "App Security & Load Balancing", smMeta.Subcategory)
	}
	if smMeta.APIURL != "https://docs.cloud.f5.com/docs-v2/api/views-securemesh-site-v2" {
		t.Fatalf("expected longest-prefix parent securemesh_site_v2 API URL, got %q", smMeta.APIURL)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sampleDoc = "---\npage_title: \"x\"\ndescription: |-\n  x\n---\n\n# f5xc_origin_pool (Resource)\n\nManages an origin pool.\n"

// TestMain_PostProcessesDocsTree verifies that main updates resource docs with
// frontmatter, skips directories, non-Markdown files, and files without
// frontmatter, reports unreadable docs, and continues when a docs target is
// not a directory.
func TestMain_PostProcessesDocsTree(t *testing.T) {
	root := t.TempDir()
	tools := filepath.Join(root, "tools")
	writeFile(t, filepath.Join(tools, "categories.yaml"), "resources:\n  origin_pool:\n    subcategory: Pools\n    api_url: https://example.invalid/op\n")
	writeFile(t, filepath.Join(tools, "docs_links.generated.yaml"), "resources: {}\n")

	resources := filepath.Join(root, "docs", "resources")
	writeFile(t, filepath.Join(resources, "origin_pool.md"), sampleDoc)
	writeFile(t, filepath.Join(resources, "plain.md"), "no frontmatter\n")
	writeFile(t, filepath.Join(resources, "notes.txt"), "ignored\n")
	// A dangling symlink fails to read regardless of the user's privileges.
	if err := os.Symlink(filepath.Join(root, "absent"), filepath.Join(resources, "dangling.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(resources, "subdir.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A regular file in place of the data-sources directory makes ReadDir fail
	// with an error other than "not exist".
	writeFile(t, filepath.Join(root, "docs", "data-sources"), "not a dir")

	t.Chdir(tools)
	main()

	got, err := os.ReadFile(filepath.Join(resources, "origin_pool.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`subcategory: "Pools"`, "- [API documentation](https://example.invalid/op)"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("updated doc does not contain %q:\n%s", want, got)
		}
	}
}

// TestMain_MissingDocsDirectories verifies that main succeeds when neither
// docs directory exists.
func TestMain_MissingDocsDirectories(t *testing.T) {
	tools := filepath.Join(t.TempDir(), "tools")
	writeFile(t, filepath.Join(tools, "categories.yaml"), "default_category: General\n")

	t.Chdir(tools)
	main()
}

// TestLoadLinksConfig_Errors verifies that read errors other than "not exist"
// and invalid YAML are returned, and that empty sections become empty maps.
func TestLoadLinksConfig_Errors(t *testing.T) {
	dir := t.TempDir()

	if _, err := loadLinksConfig(dir); err == nil {
		t.Error("expected error when reading a directory")
	}

	bad := filepath.Join(dir, "bad.yaml")
	writeFile(t, bad, "resources: [\n")
	if _, err := loadLinksConfig(bad); err == nil {
		t.Error("expected error for invalid YAML")
	}

	// Explicit null sections reset the preset maps to nil during decoding.
	empty := filepath.Join(dir, "empty.yaml")
	writeFile(t, empty, "resources: null\ndata_sources: null\n")
	links, err := loadLinksConfig(empty)
	if err != nil {
		t.Fatal(err)
	}
	if links.Resources == nil || links.DataSources == nil {
		t.Errorf("expected non-nil maps, got %+v", links)
	}
}

// TestMergeLinksConfig_NilInputsAndDataSources verifies that nil inputs act as
// empty configs and that data source overrides replace base values.
func TestMergeLinksConfig_NilInputsAndDataSources(t *testing.T) {
	if got := mergeLinksConfig(nil, nil); len(got.Resources) != 0 || len(got.DataSources) != 0 {
		t.Errorf("mergeLinksConfig(nil, nil) = %+v, want empty", got)
	}

	merged := mergeLinksConfig(
		&LinksConfig{DataSources: map[string]string{"a": "base", "b": "keep"}},
		&LinksConfig{DataSources: map[string]string{"a": "override"}},
	)
	if merged.DataSources["a"] != "override" || merged.DataSources["b"] != "keep" {
		t.Errorf("unexpected data sources: %+v", merged.DataSources)
	}
}

// TestLoadConfig_ErrorsAndDefaults verifies that a missing file and invalid
// YAML return errors, and that an empty file yields non-nil maps and the
// "General" default category.
func TestLoadConfig_ErrorsAndDefaults(t *testing.T) {
	dir := t.TempDir()

	if _, err := loadConfig(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Error("expected error for missing file")
	}

	bad := filepath.Join(dir, "bad.yaml")
	writeFile(t, bad, "resources: [\n")
	if _, err := loadConfig(bad); err == nil {
		t.Error("expected error for invalid YAML")
	}

	empty := filepath.Join(dir, "empty.yaml")
	writeFile(t, empty, "{}\n")
	cfg, err := loadConfig(empty)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Resources == nil || cfg.DataSources == nil || cfg.PrefixMappings == nil || cfg.DefaultCategory != "General" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

// TestNormalizeDocMetadataMap_APIURL verifies that an object entry keeps its
// api_url and that entries of other types are dropped.
func TestNormalizeDocMetadataMap_APIURL(t *testing.T) {
	got := normalizeDocMetadataMap(map[string]interface{}{
		"a": map[string]interface{}{"api_url": "https://example.invalid/a"},
		"b": 42,
	})

	if got["a"].APIURL != "https://example.invalid/a" {
		t.Errorf("APIURL = %q", got["a"].APIURL)
	}
	if _, ok := got["b"]; ok {
		t.Error("expected unsupported entry to be dropped")
	}
}

// TestResolveAttachmentParentAPIURL_Precedence verifies that empty and
// self-referencing entries are skipped and that the longest matching parent
// wins across links and config.
func TestResolveAttachmentParentAPIURL_Precedence(t *testing.T) {
	name := "http_loadbalancer_policy_attachments"
	links := &LinksConfig{Resources: map[string]string{
		name:   "https://example.invalid/self",
		"http": "https://example.invalid/short",
		"skip": " ",
	}}
	cfg := &Config{Resources: map[string]DocMetadata{
		name:                {APIURL: "https://example.invalid/self"},
		"empty":             {},
		"http_loadbalancer": {APIURL: "https://example.invalid/long"},
	}}

	if got := resolveAttachmentParentAPIURL(name, cfg, links); got != "https://example.invalid/long" {
		t.Errorf("resolveAttachmentParentAPIURL() = %q", got)
	}
}

// TestResolveCategory_Precedence verifies the lookup order: resource entry,
// data source entry, prefix mapping, configured default, then "General".
func TestResolveCategory_Precedence(t *testing.T) {
	cfg := &Config{
		Resources:      map[string]DocMetadata{"r": {Subcategory: "R"}},
		DataSources:    map[string]DocMetadata{"d": {Subcategory: "D"}},
		PrefixMappings: map[string]string{"pre_": "P"},
	}

	tests := []struct{ name, def, want string }{
		{"r", "", "R"},
		{"d", "", "D"},
		{"pre_x", "", "P"},
		{"other", "Def", "Def"},
		{"other", "", "General"},
	}
	for _, tt := range tests {
		cfg.DefaultCategory = tt.def
		if got := resolveCategory(tt.name, cfg); got != tt.want {
			t.Errorf("resolveCategory(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestUpdateDocFile_SkipsAndErrors verifies that missing or malformed
// frontmatter leaves the file unchanged, that a missing description appends
// the subcategory, and that read and write errors are returned.
func TestUpdateDocFile_SkipsAndErrors(t *testing.T) {
	dir := t.TempDir()
	meta := DocMetadata{Subcategory: "S"}

	if _, err := updateDocFile(filepath.Join(dir, "missing.md"), meta, false); err == nil {
		t.Error("expected read error")
	}

	for name, content := range map[string]string{
		"short.md":      "---\n",
		"unclosed.md":   "---\na: b\nc: d\n",
		"crlf_short.md": "---\r\n",
	} {
		p := filepath.Join(dir, name)
		writeFile(t, p, content)
		if updated, err := updateDocFile(p, meta, false); err != nil || updated {
			t.Errorf("%s: updated=%v err=%v, want false, nil", name, updated, err)
		}
	}

	noDesc := filepath.Join(dir, "nodesc.md")
	writeFile(t, noDesc, "---\npage_title: x\n---\nbody\n")
	if updated, err := updateDocFile(noDesc, meta, false); err != nil || !updated {
		t.Fatalf("updated=%v err=%v, want true, nil", updated, err)
	}
	got, err := os.ReadFile(noDesc)
	if err != nil {
		t.Fatalf("read %s: %v", noDesc, err)
	}
	if !strings.Contains(string(got), "page_title: x\nsubcategory: \"S\"\n---") {
		t.Errorf("subcategory not appended:\n%s", got)
	}

	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions, so the write error cannot be produced")
	}
	readOnly := filepath.Join(dir, "readonly.md")
	writeFile(t, readOnly, "---\npage_title: x\n---\nbody\n")
	if err := os.Chmod(readOnly, 0o444); err != nil {
		t.Fatal(err)
	}
	if _, err := updateDocFile(readOnly, meta, false); err == nil {
		t.Error("expected write error for read-only file")
	}
}

// TestUpsertAPIReference_NoChange verifies that the body is unchanged when the
// URL is empty, no title exists, or the title has no description.
func TestUpsertAPIReference_NoChange(t *testing.T) {
	for _, tt := range []struct{ body, url string }{
		{"# T\n\ndesc\n", " "},
		{"no title\n", "https://example.invalid"},
		{"# T\n\n", "https://example.invalid"},
	} {
		if got := upsertAPIReference(tt.body, tt.url); got != tt.body {
			t.Errorf("upsertAPIReference(%q) = %q, want unchanged", tt.body, got)
		}
	}
}
