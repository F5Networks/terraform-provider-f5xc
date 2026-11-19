//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents categories.yaml used to post-process generated docs.
type Config struct {
	DefaultCategory string                 `yaml:"default_category"`
	PrefixMappings  map[string]string      `yaml:"prefix_mappings"`
	Resources       map[string]DocMetadata `yaml:"resources"`
	DataSources     map[string]DocMetadata `yaml:"data_sources"`
}

// DocMetadata carries category and API reference URL for a docs item.
type DocMetadata struct {
	Subcategory string `yaml:"subcategory"`
	APIURL      string `yaml:"api_url"`
}

type rawConfig struct {
	DefaultCategory string                 `yaml:"default_category"`
	PrefixMappings  map[string]string      `yaml:"prefix_mappings"`
	Resources       map[string]interface{} `yaml:"resources"`
	DataSources     map[string]interface{} `yaml:"data_sources"`
}

// LinksConfig represents generated API link overrides from Omnigen render.
type LinksConfig struct {
	Resources   map[string]string `yaml:"resources"`
	DataSources map[string]string `yaml:"data_sources"`
}

type docsTarget struct {
	Dir        string
	Kind       string
	ConfigPath string
}

func main() {
	configPath := "categories.yaml"
	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Error reading %s: %v\n", configPath, err)
		os.Exit(1)
	}

	generatedLinksPath := "docs_links.generated.yaml"
	generatedLinks, err := loadLinksConfig(generatedLinksPath)
	if err != nil {
		fmt.Printf("❌ Error reading %s: %v\n", generatedLinksPath, err)
		os.Exit(1)
	}

	overrideLinksPath := "docs_link_override.yaml"
	overrideLinks, err := loadLinksConfig(overrideLinksPath)
	if err != nil {
		fmt.Printf("❌ Error reading %s: %v\n", overrideLinksPath, err)
		os.Exit(1)
	}

	links := mergeLinksConfig(generatedLinks, overrideLinks)

	targets := []docsTarget{
		{Dir: "../docs/resources", Kind: "resources", ConfigPath: configPath},
		{Dir: "../docs/data-sources", Kind: "data-sources", ConfigPath: configPath},
	}

	updatedCount := 0

	for _, target := range targets {
		entries, err := os.ReadDir(target.Dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Printf("⚠️ Directory not found or unreadable %s: %v\n", target.Dir, err)
			continue
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			docName := strings.TrimSuffix(entry.Name(), ".md")
			metadata := resolveMetadata(docName, target.Kind, cfg, links)

			filePath := filepath.Join(target.Dir, entry.Name())
			updated, err := updateDocFile(filePath, metadata, target.Kind == "resources")
			if err != nil {
				fmt.Printf("⚠️ Error updating %s: %v\n", filePath, err)
			} else if updated {
				updatedCount++
			}
		}
	}

	fmt.Printf("✅ Successfully post-processed %d documentation files.\n", updatedCount)
}

// loadLinksConfig loads generated docs link overrides. Missing file is valid.
func loadLinksConfig(configPath string) (*LinksConfig, error) {
	links := &LinksConfig{
		Resources:   map[string]string{},
		DataSources: map[string]string{},
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return links, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, links); err != nil {
		return nil, err
	}

	if links.Resources == nil {
		links.Resources = map[string]string{}
	}
	if links.DataSources == nil {
		links.DataSources = map[string]string{}
	}

	return links, nil
}

func mergeLinksConfig(base, override *LinksConfig) *LinksConfig {
	if base == nil {
		base = &LinksConfig{Resources: map[string]string{}, DataSources: map[string]string{}}
	}
	if override == nil {
		override = &LinksConfig{Resources: map[string]string{}, DataSources: map[string]string{}}
	}

	merged := &LinksConfig{
		Resources:   map[string]string{},
		DataSources: map[string]string{},
	}

	for key, value := range base.Resources {
		merged.Resources[key] = value
	}
	for key, value := range override.Resources {
		merged.Resources[key] = value
	}

	for key, value := range base.DataSources {
		merged.DataSources[key] = value
	}
	for key, value := range override.DataSources {
		merged.DataSources[key] = value
	}

	return merged
}

// loadConfig loads categories.yaml and supports both string and object entries.
func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	raw := &rawConfig{}
	if err := yaml.Unmarshal(data, raw); err != nil {
		return nil, err
	}

	cfg := &Config{
		DefaultCategory: raw.DefaultCategory,
		PrefixMappings:  raw.PrefixMappings,
		Resources:       normalizeDocMetadataMap(raw.Resources),
		DataSources:     normalizeDocMetadataMap(raw.DataSources),
	}
	if cfg.Resources == nil {
		cfg.Resources = map[string]DocMetadata{}
	}
	if cfg.DataSources == nil {
		cfg.DataSources = map[string]DocMetadata{}
	}
	if cfg.PrefixMappings == nil {
		cfg.PrefixMappings = map[string]string{}
	}
	if strings.TrimSpace(cfg.DefaultCategory) == "" {
		cfg.DefaultCategory = "General"
	}

	return cfg, nil
}

func normalizeDocMetadataMap(raw map[string]interface{}) map[string]DocMetadata {
	if len(raw) == 0 {
		return map[string]DocMetadata{}
	}

	out := make(map[string]DocMetadata, len(raw))
	for name, value := range raw {
		switch typed := value.(type) {
		case string:
			out[name] = DocMetadata{Subcategory: typed}
		case map[string]interface{}:
			entry := DocMetadata{}
			if subcategory, ok := typed["subcategory"].(string); ok {
				entry.Subcategory = subcategory
			}
			if apiURL, ok := typed["api_url"].(string); ok {
				entry.APIURL = apiURL
			}
			out[name] = entry
		}
	}

	return out
}

const attachmentDefaultSubcategory = "App Security & Load Balancing"

// resolveMetadata resolves category and API URL for a docs item.
//
// Why: Each Terraform resource and data source documentation page needs both a
// registry subcategory and an F5 XC API documentation reference URL.
//
// How: It looks up the base metadata from cfg, applies any link override from
// links, falls back to the parent target resource's API URL for synthetic
// *_attachments resources when no explicit URL is configured, and resolves the
// subcategory via resolveCategory when unset.
//
// Parameters:
//   - name:  short resource or data source name without the "f5xc_" prefix
//   - kind:  documentation kind ("resources" or "data-sources")
//   - cfg:   loaded category/metadata configuration
//   - links: optional generated and overridden API documentation links
//
// Returns:
//   - DocMetadata: resolved subcategory and API documentation URL
func resolveMetadata(name, kind string, cfg *Config, links *LinksConfig) DocMetadata {
	entry := DocMetadata{}
	if kind == "data-sources" {
		entry = cfg.DataSources[name]
		if links != nil {
			if url := strings.TrimSpace(links.DataSources[name]); url != "" {
				entry.APIURL = url
			}
		}
	} else {
		entry = cfg.Resources[name]
		if links != nil {
			if url := strings.TrimSpace(links.Resources[name]); url != "" {
				entry.APIURL = url
			}
		}
		if strings.TrimSpace(entry.APIURL) == "" && strings.HasSuffix(name, "_attachments") {
			entry.APIURL = resolveAttachmentParentAPIURL(name, cfg, links)
		}
	}

	if strings.TrimSpace(entry.Subcategory) == "" {
		entry.Subcategory = resolveCategory(name, cfg)
	}

	return entry
}

// resolveAttachmentParentAPIURL finds the parent resource's API documentation URL for an attachment resource.
//
// Why: Synthetic *_attachments resources manage policy lists on a parent target
// resource (e.g. tcp_loadbalancer or securemesh_site_v2) and do not have a
// standalone OpenAPI specification. When no explicit link override exists, they
// should inherit the parent resource's API documentation URL rather than the
// generic root URL.
//
// How: It searches links.Resources (highest precedence) and cfg.Resources for
// the longest resource key `parent` where `name` starts with `parent + "_"` and
// has a non-empty API URL.
//
// Parameters:
//   - name:  attachment resource name without the "f5xc_" prefix
//   - cfg:   loaded category/metadata configuration
//   - links: optional generated and overridden API documentation links
//
// Returns:
//   - string: the parent resource's API URL, or empty string if no parent match is found
//
// Example:
//
//	url := resolveAttachmentParentAPIURL("tcp_loadbalancer_service_policy_attachments", cfg, links)
//	// url == "https://docs.cloud.f5.com/docs-v2/api/views-tcp-loadbalancer"
func resolveAttachmentParentAPIURL(name string, cfg *Config, links *LinksConfig) string {
	bestParentLen := 0
	bestURL := ""

	if links != nil {
		for parent, rawURL := range links.Resources {
			url := strings.TrimSpace(rawURL)
			if url == "" || parent == name {
				continue
			}
			if strings.HasPrefix(name, parent+"_") && len(parent) > bestParentLen {
				bestParentLen = len(parent)
				bestURL = url
			}
		}
	}

	if cfg != nil {
		for parent, meta := range cfg.Resources {
			url := strings.TrimSpace(meta.APIURL)
			if url == "" || parent == name {
				continue
			}
			if strings.HasPrefix(name, parent+"_") && len(parent) > bestParentLen {
				bestParentLen = len(parent)
				bestURL = url
			}
		}
	}

	return bestURL
}

// resolveCategory finds the best matching category for a docs item.
//
// Why: Every documentation page requires a non-empty subcategory in its YAML
// frontmatter so the Terraform Registry groups resources consistently.
//
// How: It checks explicit resource and data-source entries first, assigns all
// *_attachments resources to "App Security & Load Balancing", then checks
// prefix mappings and the configured default category.
//
// Parameters:
//   - name: short resource or data source name without the "f5xc_" prefix
//   - cfg:  loaded category configuration
//
// Returns:
//   - string: resolved category name
func resolveCategory(name string, cfg *Config) string {
	if entry, ok := cfg.Resources[name]; ok && strings.TrimSpace(entry.Subcategory) != "" {
		return entry.Subcategory
	}
	if entry, ok := cfg.DataSources[name]; ok && strings.TrimSpace(entry.Subcategory) != "" {
		return entry.Subcategory
	}

	if strings.HasSuffix(name, "_attachments") {
		return attachmentDefaultSubcategory
	}

	for prefix, cat := range cfg.PrefixMappings {
		if strings.HasPrefix(name, prefix) {
			return cat
		}
	}

	if cfg.DefaultCategory != "" {
		return cfg.DefaultCategory
	}
	return "General"
}

// updateDocFile updates YAML frontmatter and optionally updates API Reference section.
func updateDocFile(filePath string, metadata DocMetadata, shouldUpdateAPIReference bool) (bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	strContent := string(content)

	// File must start with YAML frontmatter delimiter line "---".
	if !strings.HasPrefix(strContent, "---\n") && !strings.HasPrefix(strContent, "---\r\n") {
		return false, nil
	}

	lines := strings.Split(strContent, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return false, nil
	}

	closingIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closingIdx = i
			break
		}
	}
	if closingIdx == -1 {
		return false, nil
	}

	frontmatter := strings.Join(lines[1:closingIdx], "\n")
	body := strings.Join(lines[closingIdx+1:], "\n")

	newSubcategoryLine := fmt.Sprintf("subcategory: %q", metadata.Subcategory)

	frontmatterLines := strings.Split(frontmatter, "\n")
	filteredFrontmatterLines := make([]string, 0, len(frontmatterLines)+1)
	inserted := false
	for _, line := range frontmatterLines {
		if strings.HasPrefix(strings.TrimSpace(line), "subcategory:") {
			continue
		}

		if !inserted && strings.HasPrefix(strings.TrimSpace(line), "description:") {
			filteredFrontmatterLines = append(filteredFrontmatterLines, newSubcategoryLine)
			inserted = true
		}

		filteredFrontmatterLines = append(filteredFrontmatterLines, line)
	}
	if !inserted {
		filteredFrontmatterLines = append(filteredFrontmatterLines, newSubcategoryLine)
	}
	frontmatter = strings.Join(filteredFrontmatterLines, "\n")

	if shouldUpdateAPIReference {
		body = upsertAPIReference(body, metadata.APIURL)
	}

	newContent := "---\n" + frontmatter + "\n---\n" + body

	if !bytes.Equal(content, []byte(newContent)) {
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// upsertAPIReference inserts or updates an API reference line below the top description.
func upsertAPIReference(body, apiURL string) string {
	if strings.TrimSpace(apiURL) == "" {
		return body
	}

	lines := strings.Split(body, "\n")
	apiLine := fmt.Sprintf("- [API documentation](%s)", apiURL)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "**API Reference:**") ||
			strings.HasPrefix(trimmed, "- [API documentation](") ||
			strings.HasPrefix(trimmed, "* [API documentation](") {
			lines[i] = apiLine
			return strings.Join(lines, "\n")
		}
	}

	titleIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			titleIdx = i
			break
		}
	}
	if titleIdx == -1 {
		return body
	}

	descStart := titleIdx + 1
	for descStart < len(lines) && strings.TrimSpace(lines[descStart]) == "" {
		descStart++
	}
	if descStart >= len(lines) {
		return body
	}

	descEnd := descStart
	for descEnd < len(lines) && strings.TrimSpace(lines[descEnd]) != "" {
		descEnd++
	}

	updated := make([]string, 0, len(lines)+2)
	updated = append(updated, lines[:descEnd]...)
	updated = append(updated, apiLine)
	if descEnd < len(lines) && strings.TrimSpace(lines[descEnd]) != "" {
		updated = append(updated, "")
	}
	updated = append(updated, lines[descEnd:]...)

	return strings.Join(updated, "\n")
}
