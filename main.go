package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
)

type role struct{ name, work string }
type stopFile struct {
	goal  string
	roles []role
}

func readSTOP(text string) (stopFile, error) {
	var file stopFile
	var section, name string
	var body strings.Builder
	finish := func() error {
		work := strings.TrimSpace(body.String())
		if section == "" {
			return nil
		}
		if work == "" {
			return fmt.Errorf("%s needs text", section)
		}
		if section == "GOAL" {
			file.goal = work
		} else {
			file.roles = append(file.roles, role{name, work})
		}
		body.Reset()
		return nil
	}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "# ") {
			if err := finish(); err != nil {
				return stopFile{}, err
			}
			switch {
			case line == "# GOAL" && file.goal == "" && section == "":
				section, name = "GOAL", ""
			case strings.HasPrefix(line, "# ROLE ") && file.goal != "":
				name = strings.TrimSpace(strings.TrimPrefix(line, "# ROLE "))
				if name == "" {
					return stopFile{}, errors.New("ROLE needs a name")
				}
				section = "ROLE"
			default:
				return stopFile{}, fmt.Errorf("unexpected heading %q", line)
			}
			continue
		}
		if section == "" && strings.TrimSpace(line) != "" {
			return stopFile{}, errors.New("start with # GOAL")
		}
		body.WriteString(line + "\n")
	}
	if err := finish(); err != nil {
		return stopFile{}, err
	}
	if file.goal == "" || len(file.roles) == 0 {
		return stopFile{}, errors.New("write one GOAL and at least one ROLE")
	}
	return file, nil
}

type agentFlags map[string]string

func (a *agentFlags) String() string { return fmt.Sprint(map[string]string(*a)) }
func (a *agentFlags) Set(s string) error {
	name, path, ok := strings.Cut(s, "=")
	if !ok || name == "" || path == "" {
		return errors.New("use -agent ROLE=/path/to/executable")
	}
	if *a == nil {
		*a = agentFlags{}
	}
	(*a)[name] = path
	return nil
}

func ask(ctx context.Context, dir string, file stopFile, r role, agent, model string) (string, error) {
	message := "Work toward this GOAL through your ROLE. Give a useful result and say what remains uncertain.\n\n" +
		"GOAL\n" + file.goal + "\n\nROLE " + r.name + "\n" + r.work + "\n"
	var cmd *exec.Cmd
	if agent == "" {
		args := []string{"exec", "--ephemeral", "--sandbox", "read-only", "--skip-git-repo-check"}
		if model != "" {
			args = append(args, "--model", model)
		}
		cmd = exec.CommandContext(ctx, "codex", append(args, "-")...)
	} else {
		cmd = exec.CommandContext(ctx, agent)
	}
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(message)
	cmd.Env = append(os.Environ(), "STOP_ROLE="+r.name, "STOP_GOAL="+file.goal)
	output, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return "", fmt.Errorf("%s: %s", err, strings.TrimSpace(string(exit.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func main() {
	var agents agentFlags
	flag.Var(&agents, "agent", "ROLE=/path/to/executable (repeatable)")
	flag.Usage = func() { fmt.Fprintln(os.Stderr, "usage: stop [-agent ROLE=/path/to/executable] FILE.stop.md") }
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	path, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	file, err := readSTOP(string(data))
	if err != nil {
		fatal(err)
	}
	for name := range agents {
		found := false
		for _, r := range file.roles {
			found = found || name == r.name
		}
		if !found {
			fatal(fmt.Errorf("no ROLE named %q", name))
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	type result struct {
		text string
		err  error
	}
	results := make([]result, len(file.roles))
	var wg sync.WaitGroup
	for i, r := range file.roles {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i].text, results[i].err = ask(ctx, filepath.Dir(path), file, r, agents[r.name], os.Getenv("STOP_MODEL"))
		}()
	}
	wg.Wait()
	fmt.Printf("# GOAL\n\n%s\n", file.goal)
	failed := false
	for i, r := range file.roles {
		fmt.Printf("\n# ROLE %s\n\n", r.name)
		if results[i].err != nil || results[i].text == "" {
			if results[i].err != nil {
				fmt.Fprintf(os.Stderr, "%s failed: %v\n", r.name, results[i].err)
			} else {
				fmt.Fprintf(os.Stderr, "%s returned no text\n", r.name)
			}
			failed = true
			continue
		}
		fmt.Println(results[i].text)
	}
	if failed {
		os.Exit(1)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
