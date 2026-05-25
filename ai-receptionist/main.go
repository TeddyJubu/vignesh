package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/receptionist"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types/events"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	configPath := envOr("CONFIG_PATH", "config.json")
	promptPath := envOr("PROMPT_PATH", "prompt.txt")
	whatsmeowDB := envOr("WHATSMEOW_DB", "whatsmeow.db")
	appDB := envOr("APP_DB", "database.db")

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read prompt:", err)
		os.Exit(1)
	}
	promptTpl := string(promptBytes)

	aiClient, err := ai.NewClient(cfg.AIProvider, cfg.Model)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	appStore, err := store.Open(appDB)
	if err != nil {
		fmt.Fprintln(os.Stderr, "app db:", err)
		os.Exit(1)
	}
	defer appStore.Close()

	ctx := context.Background()
	var handler *receptionist.Handler

	waClient, err := whatsapp.New(ctx, whatsmeowDB, func(v *events.Message) {
		if handler != nil {
			handler.HandleMessage(context.Background(), v)
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "whatsapp:", err)
		os.Exit(1)
	}

	handler = receptionist.New(cfg, appStore, aiClient, waClient, promptTpl)

	if err := waClient.Start(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "start:", err)
		os.Exit(1)
	}

	fmt.Printf("%s running mode=%s model=%s owner=%s groups=%v\n",
		cfg.BusinessName, cfg.Mode, cfg.Model, cfg.OwnerNumber, cfg.ReplyToGroups)
	if waClient.WM.Store.ID == nil {
		fmt.Println("Waiting for QR scan...")
	} else {
		fmt.Println("Session linked — listening for messages")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("Shutting down...")
	waClient.Disconnect()
}
