package tools

import (
	"encoding/json"
	"fmt"
	"time"
)

type PackageMetadata struct {
	Package           string    `json:"package"`
	Version           string    `json:"version"`
	PublishedAt       time.Time `json:"published_at"`
	Publisher         string    `json:"publisher"`
	HasProvenance     bool      `json:"has_provenance"`
	HasInstallScripts bool      `json:"has_install_scripts"`
	TarballSize       int64     `json:"tarball_size"`
	FileCount         int       `json:"file_count"`
	MaintainerCount   int       `json:"maintainer_count"`
	Maintainers       []string  `json:"maintainers"`
	Anomalies         []string  `json:"anomalies,omitempty"`
}

// CheckPackageMetadata fetches npm registry metadata and checks for supply chain red flags.
func CheckPackageMetadata(pkg, version string) (*PackageMetadata, error) {
	// Fetch full package metadata
	url := fmt.Sprintf("https://registry.npmjs.org/%s", pkg)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("npm registry returned %d", resp.StatusCode)
	}

	var registry struct {
		Time        map[string]string `json:"time"`
		Maintainers []struct {
			Name string `json:"name"`
		} `json:"maintainers"`
		Versions map[string]struct {
			Scripts map[string]string `json:"scripts"`
			Dist    struct {
				Tarball    string `json:"tarball"`
				FileCount  int    `json:"fileCount"`
				UnpackSize int64  `json:"unpackedSize"`
				Signatures []struct {
					Sig string `json:"sig"`
				} `json:"signatures"`
				Attestations *struct {
					URL string `json:"url"`
				} `json:"attestations"`
			} `json:"dist"`
		} `json:"versions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		return nil, fmt.Errorf("parsing npm metadata: %w", err)
	}

	meta := &PackageMetadata{
		Package:         pkg,
		Version:         version,
		MaintainerCount: len(registry.Maintainers),
	}

	for _, m := range registry.Maintainers {
		meta.Maintainers = append(meta.Maintainers, m.Name)
	}

	// Parse publish time
	if t, ok := registry.Time[version]; ok {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			meta.PublishedAt = parsed
		}
	}

	// Check version-specific data
	if vData, ok := registry.Versions[version]; ok {
		// Install scripts (supply chain attack vector)
		for _, script := range []string{"preinstall", "install", "postinstall"} {
			if _, has := vData.Scripts[script]; has {
				meta.HasInstallScripts = true
				break
			}
		}

		meta.TarballSize = vData.Dist.UnpackSize
		meta.FileCount = vData.Dist.FileCount
		meta.HasProvenance = vData.Dist.Attestations != nil
	}

	// Detect anomalies
	meta.Anomalies = detectAnomalies(meta, registry.Time)

	return meta, nil
}

func detectAnomalies(meta *PackageMetadata, times map[string]string) []string {
	var anomalies []string

	// Check if published very recently (< 24 hours)
	if !meta.PublishedAt.IsZero() && time.Since(meta.PublishedAt) < 24*time.Hour {
		anomalies = append(anomalies, "published less than 24 hours ago")
	}

	// Check for install scripts (common attack vector)
	if meta.HasInstallScripts {
		anomalies = append(anomalies, "has install lifecycle scripts")
	}

	// Check if no provenance
	if !meta.HasProvenance {
		anomalies = append(anomalies, "no provenance attestation")
	}

	// Check for rapid version publishing (multiple versions in short time)
	if len(times) > 2 {
		var recent int
		for _, t := range times {
			if parsed, err := time.Parse(time.RFC3339, t); err == nil {
				if time.Since(parsed) < 1*time.Hour {
					recent++
				}
			}
		}
		if recent > 3 {
			anomalies = append(anomalies, fmt.Sprintf("%d versions published in the last hour (suspicious burst)", recent))
		}
	}

	return anomalies
}
