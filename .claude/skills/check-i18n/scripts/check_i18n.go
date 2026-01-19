package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	cwd, _ := os.Getwd()

	root := cwd
	if filepath.Base(cwd) == "scripts" {

		root = filepath.Dir(cwd)
	}

	keysGo := filepath.Join(root, "internal/i18n/keys.go")
	enToml := filepath.Join(root, "internal/i18n/locales/en.toml")
	zhToml := filepath.Join(root, "internal/i18n/locales/zh-CN.toml")

	keys, err := extractKeysFromGo(keysGo)
	if err != nil {
		fmt.Printf("Error extracting keys from Go (%s): %v\n", keysGo, err)
		return
	}

	enTomlKeys, err := extractKeysFromToml(enToml)
	if err != nil {
		fmt.Printf("Error extracting keys from en.toml (%s): %v\n", enToml, err)
		return
	}

	zhTomlKeys, err := extractKeysFromToml(zhToml)
	if err != nil {
		fmt.Printf("Error extracting keys from zh-CN.toml (%s): %v\n", zhToml, err)
		return
	}

	hasError := false

	fmt.Println("Comparing keys.go with en.toml:")
	if compare(keys, enTomlKeys) {
		hasError = true
	}

	fmt.Println("\nComparing keys.go with zh-CN.toml:")
	if compare(keys, zhTomlKeys) {
		hasError = true
	}

	if hasError {
		os.Exit(1)
	}
}

func extractKeysFromGo(path string) (map[string]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	keys := make(map[string]bool)
	re := regexp.MustCompile(`=\s+"([^"]+)"`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			keys[matches[1]] = true
		}
	}
	return keys, scanner.Err()
}

func extractKeysFromToml(path string) (map[string]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	keys := make(map[string]bool)
	re := regexp.MustCompile(`^\[([^\]]+)\]`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			keys[matches[1]] = true
		}
	}
	return keys, scanner.Err()
}

func compare(goKeys, tomlKeys map[string]bool) bool {
	diffFound := false

	missing := []string{}
	for k := range goKeys {
		if !tomlKeys[k] {
			missing = append(missing, k)
		}
	}

	extra := []string{}
	for k := range tomlKeys {
		if !goKeys[k] {
			extra = append(extra, k)
		}
	}

	if len(missing) > 0 {
		fmt.Println("  Missing in TOML (defined in keys.go):")
		for _, k := range missing {
			fmt.Printf("    - %s\n", k)
		}
		diffFound = true
	}

	if len(extra) > 0 {
		fmt.Println("  Extra in TOML (not in keys.go):")
		for _, k := range extra {
			fmt.Printf("    - %s\n", k)
		}
		diffFound = true
	}

	if !diffFound {
		fmt.Println("  OK")
	}

	return diffFound
}
