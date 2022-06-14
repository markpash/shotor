package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/shlex"
)

// adapted from https://code.rocketnine.space/tslocum/desktop

var (
	entryHeaderAttr = []byte("[desktop entry]")
	entryTypeAttr   = []byte("type=")
	entryNameAttr   = []byte("name=")
	entryPathAttr   = []byte("path=")
	entryExecAttr   = []byte("exec=")
)

// EntryType may be Application, Link or Directory.
type entryType int

// All entry types
const (
	unknown     entryType = iota // Unspecified or unrecognized
	application                  // Execute command
	link                         // Open browser
	directory                    // Open file manager
)

var quotes = map[string]string{
	`%%`:         `%`,
	`\\\\ `:      `\\ `,
	`\\\\` + "`": `\\` + "`",
	`\\\\$`:      `\\$`,
	`\\\\(`:      `\\(`,
	`\\\\)`:      `\\)`,
	`\\\\\`:      `\\\`,
	`\\\\\\\\`:   `\\\\`,
}

// Entry represents a parsed desktop entry.
type entry struct {
	// Type is the type of the entry. It may be Application, Link or Directory.
	eType entryType
	// Name is the name of the entry.
	name string
	// Path is the directory to start in.
	path string
	// Exec is the command(s) to be executed when launched.
	exec []string
}

// ExpandExec fills keywords in the provided entry's Exec with user arguments.
func (e *entry) expandExec(args ...string) []string {
	argsJoined := strings.Join(args, " ")

	ex := e.exec
	for i, arg := range ex {
		if arg == "%F" || arg == "%f" || arg == "%U" || arg == "%u" {
			ex[i] = argsJoined
		}
	}

	return ex
}

func unquoteExec(ex string) string {
	for qs, qr := range quotes {
		ex = strings.ReplaceAll(ex, qs, qr)
	}
	return ex
}

// Parse reads and parses a .desktop file into an *Entry.
func parseDesktopEntry(content io.Reader) (*entry, error) {
	var (
		scanner         = bufio.NewScanner(content)
		scannedBytes    []byte
		scannedBytesLen int
		entry           entry
		foundHeader     bool
	)

	for scanner.Scan() {
		scannedBytes = bytes.TrimSpace(scanner.Bytes())
		scannedBytesLen = len(scannedBytes)

		// Skip empty lines and comments
		if scannedBytesLen == 0 || scannedBytes[0] == byte('#') {
			continue
		}

		// Find the start of new sections
		if scannedBytes[0] == byte('[') && foundHeader {
			break
		}

		// Find the first section header
		if scannedBytes[0] == byte('[') && !foundHeader {
			if !bytes.EqualFold(scannedBytes[:len(entryHeaderAttr)], entryHeaderAttr) {
				return nil, errors.New("section header not found")
			}
			foundHeader = true
			continue
		}

		if bytes.EqualFold(scannedBytes[:len(entryTypeAttr)], entryTypeAttr) {
			switch strings.ToLower(string(scannedBytes[len(entryTypeAttr):])) {
			case "application":
				entry.eType = application
			case "link":
				entry.eType = link
			case "directory":
				entry.eType = directory
			}
			continue
		}

		if bytes.EqualFold(scannedBytes[:len(entryNameAttr)], entryNameAttr) {
			entry.name = string(scannedBytes[len(entryNameAttr):])
			continue
		}

		if bytes.EqualFold(scannedBytes[:len(entryPathAttr)], entryPathAttr) {
			entry.path = string(scannedBytes[len(entryPathAttr):])
			continue
		}

		if bytes.EqualFold(scannedBytes[:len(entryExecAttr)], entryExecAttr) {
			splits, err := shlex.Split(unquoteExec(string(scannedBytes[len(entryExecAttr):])))
			if err != nil {
				return nil, err
			}

			entry.exec = splits
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse desktop entry: %w", err)
	}

	if !foundHeader {
		return nil, errors.New("section header not found")
	}

	return &entry, nil
}
