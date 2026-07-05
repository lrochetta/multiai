package menu

// BMAD+ management menu — port of Show-BmadMenu (code-router.ps1 L759-876).
// Detects an existing BMAD+ installation in the current directory, then
// offers install/update actions executed through npx in the foreground.
// Divergences from the PowerShell version (deliberate):
//   - every npx action asks for confirmation before executing;
//   - the npx exit code is reported (the PS version ignored it);
//   - the version string typed for a specific-version install is validated
//     to avoid passing shell-hostile characters to npx.cmd on Windows.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lrochetta/multiai/internal/cli"
)

// bmadInfo describes a BMAD+ installation detected in a directory.
type bmadInfo struct {
	Installed bool
	Version   string // empty when unknown
	Packs     []string
}

var (
	bmadVersionRe = regexp.MustCompile(`bmad_version:\s*(\S+)`)
	bmadPackRe    = regexp.MustCompile(`^[ \t]+-\s*(\S+)`)
	// npm version or dist-tag: digits, letters, dots, hyphens (0.7.5, 0.8.0-rc.1, latest).
	npmVersionRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*$`)
)

// detectBmad mirrors the PowerShell detection order: _bmad/config.yaml first,
// then package.json devDependencies["bmad-plus"], then a bare .agents/ dir.
func detectBmad(dir string) bmadInfo {
	// 1. _bmad/config.yaml — presence alone means installed, parse best-effort.
	yamlPath := filepath.Join(dir, "_bmad", "config.yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		info := bmadInfo{Installed: true}
		if data, err := os.ReadFile(yamlPath); err == nil {
			info.Version, info.Packs = parseBmadConfig(string(data))
		}
		return info
	}

	// 2. package.json with bmad-plus in devDependencies.
	if data, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil {
		var pkg struct {
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			if v, ok := pkg.DevDependencies["bmad-plus"]; ok {
				return bmadInfo{Installed: true, Version: v}
			}
		}
	}

	// 3. .agents/ directory present, version unknown.
	if fi, err := os.Stat(filepath.Join(dir, ".agents")); err == nil && fi.IsDir() {
		return bmadInfo{Installed: true}
	}

	return bmadInfo{}
}

// parseBmadConfig extracts bmad_version and the packs list from a BMAD+
// config.yaml. Line-based on purpose: it mirrors the PowerShell regexes
// without pulling a YAML dependency for two fields.
func parseBmadConfig(content string) (version string, packs []string) {
	if m := bmadVersionRe.FindStringSubmatch(content); m != nil {
		version = m[1]
	}
	inPacks := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if !inPacks {
			if strings.TrimSpace(line) == "packs:" {
				inPacks = true
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break // first non-indented line ends the packs block
		}
		if m := bmadPackRe.FindStringSubmatch(line); m != nil {
			packs = append(packs, m[1])
		}
	}
	return version, packs
}

// ShowBmadMenu displays BMAD+ status for the current directory and runs the
// selected npx action. Safe to call from the interactive loop or one-shot.
func ShowBmadMenu() {
	targetDir, err := os.Getwd()
	if err != nil {
		cli.PrintError(fmt.Sprintf("Dossier courant inaccessible : %v", err))
		return
	}
	info := detectBmad(targetDir)

	fmt.Println()
	cli.PrintInfo("BMAD+ -- Gestion du framework")
	fmt.Println(strings.Repeat("-", 58))
	fmt.Println()
	fmt.Printf("  Dossier cible : %s\n", targetDir)
	if info.Installed {
		suffix := ""
		if info.Version != "" {
			suffix = fmt.Sprintf(" (v%s)", info.Version)
		}
		cli.PrintSuccess("BMAD+ detecte" + suffix)
		if len(info.Packs) > 0 {
			fmt.Printf("  Packs installes : %s\n", strings.Join(info.Packs, ", "))
		}
	} else {
		fmt.Println("  BMAD+ non detecte dans ce dossier.")
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// npx (Node.js) is the only supported installer. LookPath resolves
	// npx.cmd on Windows via PATHEXT.
	npxPath, err := exec.LookPath("npx")
	if err != nil {
		cli.PrintWarning("npx introuvable. Node.js est requis :")
		fmt.Println("  -> https://nodejs.org")
		fmt.Println()
		fmt.Print("  Entree pour revenir : ")
		readMenuLine(reader)
		return
	}

	var npxArgs []string
	if info.Installed {
		fmt.Println("  1. Mise a jour vers la derniere version stable (latest)")
		fmt.Println("     npx bmad-plus@latest install --yes")
		fmt.Println("  2. Reinstallation complete (tous les packs)")
		fmt.Println("     npx bmad-plus install --yes --packs all")
		fmt.Println("  3. Mise a jour vers une version specifique")
		fmt.Println("     npx bmad-plus@<version> install --yes")
		fmt.Println("  4. Installation fraiche (reinitialise tout)")
		fmt.Println("     npx bmad-plus install --yes --force")
		fmt.Println("  0. Retour")
		fmt.Println()
		fmt.Print("  Choix : ")
		switch readMenuLine(reader) {
		case "1":
			npxArgs = []string{"bmad-plus@latest", "install", "--yes"}
		case "2":
			npxArgs = []string{"bmad-plus", "install", "--yes", "--packs", "all"}
		case "3":
			fmt.Print("  Version (ex: 0.7.5) : ")
			ver := readMenuLine(reader)
			if ver == "" {
				fmt.Println("  Annule.")
				return
			}
			if !npmVersionRe.MatchString(ver) {
				cli.PrintWarning("Version invalide (caracteres autorises : lettres, chiffres, points, tirets).")
				return
			}
			npxArgs = []string{"bmad-plus@" + ver, "install", "--yes"}
		case "4":
			npxArgs = []string{"bmad-plus", "install", "--yes", "--force"}
		case "0", "":
			return
		default:
			cli.PrintWarning("Choix invalide.")
			return
		}
	} else {
		fmt.Println("  1. Installation complete silencieuse (tous les packs)")
		fmt.Println("     npx bmad-plus install --yes --packs all")
		fmt.Println("  2. Installation interactive (choisir les packs)")
		fmt.Println("     npx bmad-plus install")
		fmt.Println("  3. Installation derniere version (latest)")
		fmt.Println("     npx bmad-plus@latest install --yes")
		fmt.Println("  0. Retour")
		fmt.Println()
		fmt.Print("  Choix : ")
		switch readMenuLine(reader) {
		case "1":
			npxArgs = []string{"bmad-plus", "install", "--yes", "--packs", "all"}
		case "2":
			npxArgs = []string{"bmad-plus", "install"}
		case "3":
			npxArgs = []string{"bmad-plus@latest", "install", "--yes"}
		case "0", "":
			return
		default:
			cli.PrintWarning("Choix invalide.")
			return
		}
	}

	runNpx(reader, npxPath, targetDir, npxArgs)
}

// runNpx echoes the exact command, asks for confirmation, then runs npx in
// the foreground with inherited stdio (interactive installs keep working).
func runNpx(reader *bufio.Reader, npxPath, dir string, args []string) {
	fmt.Println()
	fmt.Printf("  Commande : npx %s\n", strings.Join(args, " "))
	fmt.Print("  Executer ? (o/N) : ")
	answer := strings.ToLower(readMenuLine(reader))
	if answer != "o" && answer != "oui" && answer != "y" {
		fmt.Println("  Annule.")
		return
	}
	fmt.Println()

	cmd := exec.Command(npxPath, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	var exitErr *exec.ExitError
	switch {
	case err == nil:
		cli.PrintSuccess("npx termine (code 0)")
	case errors.As(err, &exitErr):
		cli.PrintWarning(fmt.Sprintf("npx termine avec le code %d", exitErr.ExitCode()))
	default:
		cli.PrintError(fmt.Sprintf("Echec du lancement de npx : %v", err))
	}
}

// readMenuLine reads one trimmed line; on EOF it returns whatever was read so
// exhausted piped input behaves like "cancel" instead of looping forever.
func readMenuLine(reader *bufio.Reader) string {
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}
