package audit

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const skillName = "spectre-audit"

func setupCommand(args []string, out, diagnostics io.Writer) error {
	flags := flag.NewFlagSet("spectre audit setup", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	agent := flags.String("agent", "", "codex, claude, gemini, or opencode")
	skills := flags.String("skills-dir", "", "custom personal skills directory")
	replace := flags.Bool("replace", false, "replace another installation's skill link")
	remove := flags.Bool("remove", false, "remove only this installation's skill link")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return invalid("Unexpected setup argument")
	}
	if *replace && *remove {
		return invalid("Use --replace or --remove, not both")
	}
	directories := map[string]string{"codex": ".agents/skills", "claude": ".claude/skills", "gemini": ".agents/skills", "opencode": ".agents/skills"}
	if *agent != "" {
		directory, ok := directories[*agent]
		if !ok {
			return invalid("Unknown agent %q", *agent)
		}
		if *skills == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			*skills = filepath.Join(home, directory)
		}
	}
	if *skills == "" && (*replace || *remove) {
		return invalid("--agent or --skills-dir is required when changing a registration")
	}
	if !*remove {
		if _, err := exec.LookPath("git"); err != nil {
			return fmt.Errorf("Git must be on PATH for repository fingerprints: %w", err)
		}
	}
	root, err := resourceDir()
	if err != nil {
		return err
	}
	if *skills != "" {
		directory, err := filepath.Abs(*skills)
		if err != nil {
			return err
		}
		action, err := registerSkill(root, directory, *replace, *remove)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "%s: %s\n", action, filepath.Join(directory, skillName)); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(out, "Setup checks passed. No skill registration was changed."); err != nil {
			return err
		}
	}
	if !*remove {
		_, err = fmt.Fprintf(out, "\nStart a fresh coding-agent session in your project and ask:\n\nUse spectre-audit to audit this project. Resume any existing audit first.\n\nFor any agent with local file and shell access, paste:\n\nRead and follow %q to audit the current project. Resume any existing audit first.\n\nKeep Spectre on PATH. Schemas and instructions are in %s.\n", filepath.Join(root, "SKILL.md"), root)
	}
	return err
}
func registerSkill(root, directory string, replace, remove bool) (action string, err error) {
	destination := filepath.Join(directory, skillName)
	if remove {
		if _, err := os.Lstat(destination); errors.Is(err, os.ErrNotExist) {
			return "Already removed", nil
		}
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	release, err := acquireLock(filepath.Join(directory, ".spectre-audit-setup.lock"))
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, release()) }()
	info, err := os.Lstat(destination)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	matches := false
	if exists {
		target, err := filepath.EvalSymlinks(destination)
		if err == nil {
			matches = target == root
		}
		if info.Mode()&os.ModeSymlink == 0 {
			if matches && !remove {
				return "Already installed", nil
			}
			return "", conflict("%s is an existing file or directory. Setup will not replace it.", destination)
		}
	}
	if remove {
		if !exists {
			return "Already removed", nil
		}
		if !matches {
			return "", conflict("%s points to another installation. Setup will not remove it.", destination)
		}
		return "Removed", os.Remove(destination)
	}
	if matches {
		return "Already installed", nil
	}
	if exists && !replace {
		return "", conflict("%s points elsewhere. Review it and rerun with --replace.", destination)
	}
	id, err := uuid()
	if err != nil {
		return "", err
	}
	temporary := filepath.Join(directory, ".spectre-audit-"+id)
	defer os.Remove(temporary)
	if err := os.Symlink(root, temporary); err != nil {
		return "", err
	}
	if err := os.Rename(temporary, destination); err != nil {
		return "", err
	}
	return "Installed", nil
}
