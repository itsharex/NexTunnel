package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxPayloadEntryBytes = 512 << 20

func assertPayloadHash(payload []byte, expected string) error {
	expected = strings.TrimSpace(strings.ToLower(expected))
	if expected == "" {
		return fmt.Errorf("payload SHA256 不能为空")
	}
	actualBytes := sha256.Sum256(payload)
	actual := hex.EncodeToString(actualBytes[:])
	if actual != expected {
		return fmt.Errorf("payload SHA256 校验失败：expected=%s actual=%s", expected, actual)
	}
	return nil
}

func safeExtractZip(payload []byte, destination string, report ProgressReporter) error {
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return fmt.Errorf("打开 payload zip: %w", err)
	}
	if len(reader.File) == 0 {
		return fmt.Errorf("payload zip 为空")
	}
	cleanDestination, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("解析解压目标目录: %w", err)
	}
	for index, file := range reader.File {
		if err := extractZipEntry(file, cleanDestination); err != nil {
			return err
		}
		if report != nil {
			percent := 20 + int(float64(index+1)/float64(len(reader.File))*45)
			report(InstallProgress{Phase: installPhaseExtracting, Percent: percent, Message: "正在解压应用文件"})
		}
	}
	return nil
}

func extractZipEntry(file *zip.File, destination string) error {
	if file.FileInfo().Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("payload 包含不允许的符号链接：%s", file.Name)
	}
	targetPath, err := safeJoin(destination, file.Name)
	if err != nil {
		return err
	}
	if file.FileInfo().IsDir() {
		return os.MkdirAll(targetPath, 0o755)
	}
	if file.UncompressedSize64 > maxPayloadEntryBytes {
		return fmt.Errorf("payload 条目过大：%s", file.Name)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("创建目录 %s: %w", filepath.Dir(targetPath), err)
	}
	source, err := file.Open()
	if err != nil {
		return fmt.Errorf("打开 payload 条目 %s: %w", file.Name, err)
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.FileInfo().Mode().Perm())
	if err != nil {
		return fmt.Errorf("创建文件 %s: %w", targetPath, err)
	}
	defer target.Close()
	writtenBytes, err := io.Copy(target, io.LimitReader(source, maxPayloadEntryBytes+1))
	if err != nil {
		return fmt.Errorf("写入文件 %s: %w", targetPath, err)
	}
	if writtenBytes > maxPayloadEntryBytes {
		return fmt.Errorf("payload 条目过大：%s", file.Name)
	}
	return nil
}

func safeJoin(root string, entryName string) (string, error) {
	normalizedName := filepath.Clean(filepath.FromSlash(entryName))
	if normalizedName == "." || normalizedName == "" {
		return "", fmt.Errorf("payload 包含空路径")
	}
	if filepath.IsAbs(normalizedName) || strings.HasPrefix(normalizedName, ".."+string(filepath.Separator)) || normalizedName == ".." {
		return "", fmt.Errorf("payload 包含非法路径：%s", entryName)
	}
	if strings.Contains(normalizedName, ":") {
		return "", fmt.Errorf("payload 包含非法路径：%s", entryName)
	}
	targetPath := filepath.Join(root, normalizedName)
	if !isPathInside(root, targetPath) {
		return "", fmt.Errorf("payload 路径越界：%s", entryName)
	}
	return targetPath, nil
}

func isPathInside(parent string, child string) bool {
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return false
	}
	childAbs, err := filepath.Abs(child)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(parentAbs, childAbs)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
