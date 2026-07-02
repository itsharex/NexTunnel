package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
)

//go:embed payload/*
var embeddedPayloadFiles embed.FS

type PayloadSource interface {
	Manifest() (PayloadManifest, error)
	PayloadBytes(manifest PayloadManifest) ([]byte, error)
}

type embeddedPayloadSource struct {
	files fs.FS
}

func newEmbeddedPayloadSource() embeddedPayloadSource {
	return embeddedPayloadSource{files: embeddedPayloadFiles}
}

func (s embeddedPayloadSource) Manifest() (PayloadManifest, error) {
	data, err := fs.ReadFile(s.files, "payload/manifest.json")
	if err != nil {
		return PayloadManifest{}, fmt.Errorf("读取安装 payload manifest: %w", err)
	}
	var manifest PayloadManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return PayloadManifest{}, fmt.Errorf("解析安装 payload manifest: %w", err)
	}
	applyManifestDefaults(&manifest)
	return manifest, nil
}

func (s embeddedPayloadSource) PayloadBytes(manifest PayloadManifest) ([]byte, error) {
	if manifest.PayloadFile == "" {
		return nil, fmt.Errorf("安装 payload 未配置")
	}
	payloadPath := path.Join("payload", manifest.PayloadFile)
	data, err := fs.ReadFile(s.files, payloadPath)
	if err != nil {
		return nil, fmt.Errorf("读取安装 payload %s: %w", manifest.PayloadFile, err)
	}
	return data, nil
}

func applyManifestDefaults(manifest *PayloadManifest) {
	if manifest.Version == "" {
		manifest.Version = AppVersion
	}
	if manifest.Target == "" {
		manifest.Target = "windows/amd64"
	}
	if manifest.AppExecutable == "" {
		manifest.AppExecutable = appExecutableName
	}
	if manifest.RequiredSpaceMB <= 0 {
		manifest.RequiredSpaceMB = defaultRequiredSpaceMB
	}
	if manifest.Signing == "" {
		manifest.Signing = "unsigned-alpha"
	}
}
