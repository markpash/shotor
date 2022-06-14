package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/adrg/xdg"
)

type mapping struct {
	Regex  *Regexp `json:"regex"`
	Target string  `json:"target"`
}

func parseConfig(path string) ([]mapping, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := json.NewDecoder(f)

	var ret []mapping
	if err := d.Decode(&ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func main() {
	cfgPath := flag.String("config", "~/.config/shotor/config.json", "path to configuration file")
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Fatal("incorrent number of arguments provided")
	}

	urlStr := flag.Arg(0)

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		log.Fatal("error: only support http and https schemes")
	}

	mappings, err := parseConfig(*cfgPath)
	if err != nil {
		log.Fatalf("error: failed to parse config %v:", err)
	}

	for _, mapping := range mappings {
		if mapping.Regex.MatchString(parsedURL.String()) {
			if err := launchWithDesktopEntry(mapping.Target, urlStr); err == nil {
				os.Exit(0)
			}
		}
	}

	os.Exit(1)
}

func launchWithDesktopEntry(desktopEntry string, args ...string) error {
	foundDesktopEntry, err := findDesktopEntry(desktopEntry)
	if err != nil {
		return fmt.Errorf("couldn't open desktop entry file: %w", err)
	}

	entry, err := parseDesktopEntry(foundDesktopEntry)
	if err != nil {
		return fmt.Errorf("couldn't parse desktop entry file: %w", err)
	}

	args = entry.expandExec(args...)

	absBinPath, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("couldn't find executable %s in $PATH: %w", args[0], err)
	}

	if _, err := syscall.ForkExec(absBinPath, args, &syscall.ProcAttr{
		Env: os.Environ(),
	}); err != nil {
		return fmt.Errorf("failed to exec desktop entry: %w", err)
	}

	return nil
}

func findDesktopEntry(name string) (*os.File, error) {
	for _, dir := range xdg.DataDirs {
		absPathDesktopEntry := path.Join(dir, "applications", name)
		if f, err := os.Open(absPathDesktopEntry); err == nil {
			return f, nil
		}
	}

	return nil, fmt.Errorf("desktop entry for %s not found", name)
}
