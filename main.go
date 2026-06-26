package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var Version string

type Task struct {
	Name     string
	Commands []string
}

type Config struct {
	Tasks map[string]Task
}

func find() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		path := filepath.Join(dir, "sarc")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no sarc found")
		}
		dir = parent
	}
}

func parse(data []byte) (*Config, error) {
	cfg := &Config{Tasks: make(map[string]Task)}
	var current string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasSuffix(trimmed, ":") {
			current = strings.TrimSuffix(trimmed, ":")
			cfg.Tasks[current] = Task{Name: current}
			continue
		}
		if len(line) > 0 && (line[0] == '\t' || line[0] == ' ') {
			if current == "" {
				continue
			}
			task := cfg.Tasks[current]
			task.Commands = append(task.Commands, strings.TrimSpace(line))
			cfg.Tasks[current] = task
		}
	}
	return cfg, nil
}

func run(cfg *Config, task Task) error {
	for _, cmd := range task.Commands {
		if strings.HasPrefix(cmd, "@") {
			name := strings.TrimPrefix(cmd, "@")
			nested, ok := cfg.Tasks[name]
			if !ok {
				return fmt.Errorf("task %q not found", name)
			}
			if err := run(cfg, nested); err != nil {
				return err
			}
			continue
		}
		c := exec.Command("sh", "-c", cmd)
		c.Stdout = os.Stdout
		c.Stdin = os.Stdin
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("task %q failed: %w", task.Name, err)
		}
	}
	return nil
}

func list(cfg *Config, sym string) {
	var names []string
	for name := range cfg.Tasks {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("%s%s\n", sym, name)
	}
}

func main() {
	path, err := find()
	if err != nil {
		fmt.Println(err)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	cfg, err := parse(data)
	if err != nil {
		fmt.Println(err)
		return
	}
	if len(os.Args) < 2 {
		list(cfg, "• ")
		return
	}
	switch os.Args[1] {
	case "--raw":
		// for future autocomplete
		list(cfg, "")
		return
	case "-v", "--version":
		fmt.Printf("sar %s\n", Version)
	default:
		task, ok := cfg.Tasks[os.Args[1]]
		if !ok {
			fmt.Printf("task \"%s\" not found\n", os.Args[1])
			return
		}
		if err := run(cfg, task); err != nil {
			fmt.Println(err)
			return
		}
	}
}
