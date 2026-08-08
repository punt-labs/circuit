package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const exitUsage = 2

type command struct {
	stdout io.Writer
	stderr io.Writer
}

func main() {
	cmd := command{stdout: os.Stdout, stderr: os.Stderr}
	if err := cmd.run(os.Args[1:]); err != nil {
		fmt.Fprintln(cmd.stderr, err)
		os.Exit(exitUsage)
	}
}

func (cmd command) run(args []string) error {
	if len(args) == 0 {
		cmd.printUsage()
		return errors.New("missing command")
	}

	switch args[0] {
	case "help", "--help", "-h":
		cmd.printUsage()
		return nil
	case "validate":
		return cmd.validate(args[1:])
	case "summary":
		return cmd.summary(args[1:])
	default:
		cmd.printUsage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func (cmd command) validate(args []string) error {
	path, err := singlePath(args)
	if err != nil {
		return err
	}
	info, err := inspectFile(path)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.stdout, "valid: %s (%d bytes)\n", info.cleanPath, info.size)
	return nil
}

func (cmd command) summary(args []string) error {
	path, err := singlePath(args)
	if err != nil {
		return err
	}
	info, err := inspectFile(path)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.stdout, "file: %s\n", info.cleanPath)
	fmt.Fprintf(cmd.stdout, "size: %d bytes\n", info.size)
	return nil
}

func (cmd command) printUsage() {
	fmt.Fprintln(cmd.stderr, "usage: circuit <command> <file>")
	fmt.Fprintln(cmd.stderr, "")
	fmt.Fprintln(cmd.stderr, "commands:")
	fmt.Fprintln(cmd.stderr, "  validate <file>  check that a playbook file is readable")
	fmt.Fprintln(cmd.stderr, "  summary <file>   print a small playbook file summary")
}

func singlePath(args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("expected exactly one file path, got %d", len(args))
	}
	return args[0], nil
}

type fileInfo struct {
	cleanPath string
	size      int64
}

func inspectFile(path string) (fileInfo, error) {
	cleanPath := filepath.Clean(path)
	stat, err := os.Stat(cleanPath)
	if err != nil {
		return fileInfo{}, fmt.Errorf("read %s: %w", cleanPath, err)
	}
	if stat.IsDir() {
		return fileInfo{}, fmt.Errorf("read %s: path is a directory", cleanPath)
	}
	return fileInfo{cleanPath: cleanPath, size: stat.Size()}, nil
}
