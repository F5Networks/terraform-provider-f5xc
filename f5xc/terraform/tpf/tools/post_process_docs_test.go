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

func TestResolveMetadata_UsesGeneratedLinksOverride(t *testing.T) {
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
		t.Fatalf("expected generated links override URL, got %q", meta.APIURL)
	}
	if meta.Subcategory != "App Security & Load Balancing" {
		t.Fatalf("expected subcategory to come from categories config, got %q", meta.Subcategory)
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
