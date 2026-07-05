package cli

import "fmt"

// CompletionScripts returns shell completion scripts for each supported shell.
var CompletionScripts = map[string]string{
	"bash": `# Bash completion for multiai
_multiai_bash() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    if [[ ${COMP_CWORD} -eq 1 ]]; then
        opts="launch list config models search compare bmad version help completion"
    elif [[ ${prev} == "launch" ]]; then
        opts="-p --profile -t --tool --json --dry-run --no-launch --show-env --allow-custom-command"
    elif [[ ${prev} == "-p" || ${prev} == "--profile" ]]; then
        opts="$(multiai list 2>/dev/null | awk 'NR>2{print $2}')"
    elif [[ ${prev} == "config" ]]; then
        opts="--provider"
    elif [[ ${prev} == "models" || ${prev} == "search" || ${prev} == "compare" ]]; then
        opts="--json --offline --top --sort --free --category --vendor"
    elif [[ ${prev} == "list" ]]; then
        opts="--json"
    fi
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
}
complete -F _multiai_bash multiai
`,

	"zsh": `#compdef multiai
_multiai_zsh() {
    local -a commands
    commands=('launch:Lancer un CLI' 'list:Lister les profils' 'config:Configurer les cles API' 'models:Modeles OpenRouter' 'search:Rechercher un modele' 'compare:Comparer des modeles' 'bmad:BMAD+' 'version:Afficher la version' 'help:Aide' 'completion:Shell completion')
    _arguments -C '1: :->command' '*:: :->args'
    case $state in
        command) _describe 'command' commands ;;
        args)
            case $words[1] in
                launch) _arguments '-p[Profil]:profile:()' '--profile[Profil]:profile:()' '--json[Sortie JSON]' '--dry-run[Simulation]' '--no-launch[Pas de lancement]' '--show-env[Voir environnement]' ;;
                config) _arguments '--provider[Fournisseur]:provider:()' ;;
                models|search|compare) _arguments '--json[Sortie JSON]' '--offline[Hors-ligne]' '--top[Top N]' '--sort[Tri]' '--free[Gratuits]' '--category[Modalite]' '--vendor[Fournisseur]' ;;
                list) _arguments '--json[Sortie JSON]' ;;
            esac ;;
    esac
}
compdef _multiai_zsh multiai
`,

	"fish": `# Fish completion for multiai
complete -c multiai -n '__fish_use_subcommand' -a launch -d 'Lancer un CLI'
complete -c multiai -n '__fish_use_subcommand' -a list -d 'Lister les profils'
complete -c multiai -n '__fish_use_subcommand' -a config -d 'Configurer les cles API'
complete -c multiai -n '__fish_use_subcommand' -a models -d 'Modeles OpenRouter'
complete -c multiai -n '__fish_use_subcommand' -a search -d 'Rechercher un modele'
complete -c multiai -n '__fish_use_subcommand' -a compare -d 'Comparer des modeles'
complete -c multiai -n '__fish_use_subcommand' -a bmad -d 'BMAD+'
complete -c multiai -n '__fish_use_subcommand' -a version -d 'Afficher la version'
complete -c multiai -n '__fish_use_subcommand' -a help -d 'Aide'
complete -c multiai -n '__fish_use_subcommand' -a completion -d 'Shell completion'
complete -c multiai -n '__fish_seen_subcommand_from config' -l provider -d 'Configurer un fournisseur'
complete -c multiai -n '__fish_seen_subcommand_from models search compare' -l json -d 'Sortie JSON'
complete -c multiai -n '__fish_seen_subcommand_from models search compare' -l offline -d 'Hors-ligne'
complete -c multiai -n '__fish_seen_subcommand_from launch' -s p -l profile -d 'Profil a lancer'
complete -c multiai -n '__fish_seen_subcommand_from launch' -l json -d 'Sortie JSON'
complete -c multiai -n '__fish_seen_subcommand_from launch' -l dry-run -d 'Simulation'
complete -c multiai -n '__fish_seen_subcommand_from launch' -l no-launch -d 'Ne pas lancer'
complete -c multiai -n '__fish_seen_subcommand_from launch' -l show-env -d 'Afficher environnement'
complete -c multiai -n '__fish_seen_subcommand_from list' -l json -d 'Sortie JSON'
`,

	"powershell": `# PowerShell completion for multiai
Register-ArgumentCompleter -Native -CommandName multiai -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $commands = @('launch', 'list', 'config', 'models', 'search', 'compare', 'bmad', 'version', 'help', 'completion')
    $launchOpts = @('-p', '--profile', '-t', '--tool', '--json', '--dry-run', '--no-launch', '--show-env', '--allow-custom-command')
    $orOpts = @('--json', '--offline', '--top', '--sort', '--free', '--category', '--vendor')
    $profiles = @(multiai list 2>$null | Select-Object -Skip 2 | ForEach-Object { ($_ -split '\s+')[1] } | Where-Object { $_ })
    $commandElements = $commandAst.CommandElements
    $command = $null
    for ($i = 1; $i -lt $commandElements.Count; $i++) {
        if ($commandElements[$i] -is [System.Management.Automation.Language.StringConstantExpressionAst]) { $command = $commandElements[$i].Value; break }
    }
    if (-not $command -or $command -eq 'multiai') { return $commands | Where-Object { $_ -like "$wordToComplete*" } }
    switch ($command) {
        'launch' {
            $prev = $null
            for ($i = $commandElements.Count - 1; $i -ge 0; $i--) {
                if ($commandElements[$i] -is [System.Management.Automation.Language.StringConstantExpressionAst]) { $prev = $commandElements[$i].Value; break }
            }
            if ($prev -in @('-p', '--profile')) { return $profiles | Where-Object { $_ -like "$wordToComplete*" } }
            return $launchOpts | Where-Object { $_ -like "$wordToComplete*" }
        }
        'config' { return @('--provider') | Where-Object { $_ -like "$wordToComplete*" } }
        'models' { return $orOpts | Where-Object { $_ -like "$wordToComplete*" } }
        'search' { return $orOpts | Where-Object { $_ -like "$wordToComplete*" } }
        'compare' { return $orOpts | Where-Object { $_ -like "$wordToComplete*" } }
        'list' { return @('--json') | Where-Object { $_ -like "$wordToComplete*" } }
    }
}
`,
}

// GenerateCompletion writes the completion script for the given shell to stdout.
func GenerateCompletion(shell string) error {
	script, ok := CompletionScripts[shell]
	if !ok {
		return fmt.Errorf("shell non supporte : %s. Options : bash, zsh, fish, powershell", shell)
	}
	fmt.Print(script)
	return nil
}
