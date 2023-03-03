package test

import (
	"bufio"
	"errors"
	"log"
	"os"
	"regexp"
	"testing"
)

func TestGoVersion(t *testing.T) {
	gomod, err := parseGoMod()
	if err != nil {
		log.Fatal(err)
	}

	dockerfile, err := parseDockerfile()
	if err != nil {
		log.Fatal(err)
	}

	if gomod != dockerfile {
		t.Errorf("Version mismatch! go.mod: %s, Dockerfile: %s", gomod, dockerfile)
	}
}

func parseDockerfile() (string, error) {
	return parse("../Dockerfile", `^FROM\sgolang:(\d\.\d+)`)
}

func parseGoMod() (string, error) {
	return parse("../go.mod", `^go\s+(\d\.\d+)$`)
}

func parse(file, regex string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	re, err := regexp.Compile(regex)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(f)
	var version string
	for scanner.Scan() {
		line := scanner.Text()
		match := re.FindStringSubmatch(line)
		if len(match) > 0 {
			version = match[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if version == "" {
		return "", errors.New("go version not found in go.mod")
	}

	return version, nil
}
