package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lrochetta/multiai/internal/bridge"
)

func init() { register("bridge", cmdBridge) }

// cmdBridge runs the embedded Anthropic->OpenAI translation proxy in the
// foreground, for use outside the launcher (e.g. pointing any Claude Code
// session at an OpenAI-compatible backend). Profiles with
// BRIDGE=anthropic-openai do NOT need this command: the launcher embeds
// the same proxy automatically on an ephemeral port.
func cmdBridge(args []string) int {
	fs := flag.NewFlagSet("bridge", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	target := fs.String("target", bridge.DefaultNvidiaTarget, "URL de base OpenAI-compatible du backend")
	keyEnv := fs.String("key-env", "NVIDIA_API_KEY", "variable d'environnement contenant la cle API du backend")
	port := fs.Int("port", 4100, "port local d'ecoute (127.0.0.1 uniquement)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage : multiai bridge [--target URL] [--key-env VAR] [--port N]")
		fmt.Fprintln(os.Stderr, "Demarre le pont Anthropic->OpenAI integre (pour Claude Code).")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	key := os.Getenv(*keyEnv)
	if key == "" {
		fmt.Fprintf(os.Stderr, "[X] La variable %s est vide. Exemple : $env:%s = \"nvapi-...\" (cle : https://build.nvidia.com/settings/api-keys)\n", *keyEnv, *keyEnv)
		return 2
	}

	srv, err := bridge.Start(bridge.Config{
		Target: *target,
		APIKey: key,
		Addr:   fmt.Sprintf("127.0.0.1:%d", *port),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] %v\n", err)
		return 1
	}

	fmt.Printf("Pont Anthropic->OpenAI : %s -> %s\n", srv.URL(), srv.Target())
	fmt.Printf("Cote client : ANTHROPIC_BASE_URL=%s (ANTHROPIC_AUTH_TOKEN : valeur libre)\n", srv.URL())
	fmt.Println("Ctrl+C pour arreter.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	srv.Shutdown()
	fmt.Println("Pont arrete.")
	return 0
}
