package envfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Read 读取简单 KEY=VALUE 环境文件，保持与 deploy/server 脚本生成格式兼容。
func Read(path string) (map[string]string, error) {
	values := map[string]string{}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open env file: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan env file: %w", err)
	}
	return values, nil
}
