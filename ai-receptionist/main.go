package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ai-receptionist/internal/adapters/calendar"
	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/httpapi"
	"ai-receptionist/internal/models"
	"ai-receptionist/internal/ops"
	"ai-receptionist/internal/pb"
	"ai-receptionist/internal/receptionist"
	"ai-receptionist/internal/settings"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/tools/composio"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types/events"
)

func loadStyleExamples() string {
	path := envOr("STYLE_EXAMPLES_PATH", "style-examples.txt")
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

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
	httpAddr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	graphitiURL := strings.TrimSpace(os.Getenv("GRAPHITI_URL"))

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

	instructionsPath := envOr("INSTRUCTIONS_PATH", "knowledge/instructions.md")
	instructionsMD := ""
	if b, err := os.ReadFile(instructionsPath); err == nil {
		instructionsMD = strings.TrimSpace(string(b))
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "read instructions:", err)
		os.Exit(1)
	}

	appStore, err := store.Open(appDB)
	if err != nil {
		fmt.Fprintln(os.Stderr, "app db:", err)
		os.Exit(1)
	}
	defer appStore.Close()

	settingResolver := settings.New(appStore)
	models.SetConfigModel(cfg.Model)
	if p, err := settingResolver.ResolvedAIProvider(); err == nil && p != "" {
		models.SetActiveProvider(p)
	} else {
		models.SetActiveProvider(cfg.ResolvedAIProvider())
	}
	models.SetSettingsModelResolver(func() string {
		if settingResolver == nil {
			return ""
		}
		p, err := settingResolver.ResolvedAIProvider()
		if err != nil || p == "" {
			return ""
		}
		var key string
		switch p {
		case "openai":
			key = "openai.model"
		case "anthropic":
			key = "anthropic.model"
		case "openrouter":
			key = "openrouter.model"
		case "custom":
			key = "custom.model"
		default:
			key = "ollama.model"
		}
		m, _ := settingResolver.Get(key)
		return m
	})
	aiClient, err := ai.NewProviderFromSettings(cfg, settingResolver)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	intentAI, err := ai.NewProviderForModel(cfg, settingResolver, models.GetModel("intent_classify"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "intent classifier provider:", err)
		os.Exit(1)
	}
	plannerAI, err := ai.NewProviderForModel(cfg, settingResolver, models.GetModel("planner"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "planner provider:", err)
		os.Exit(1)
	}
	collateAI, err := ai.NewProviderForModel(cfg, settingResolver, models.GetModel("collate"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "collate provider:", err)
		os.Exit(1)
	}
	pbClient := pb.NewFromEnv()
	pbRepo := pb.NewRepo(pbClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var handler *receptionist.Handler

	waClient, err := whatsapp.New(ctx, whatsmeowDB, func(v *events.Message) {
		if handler != nil {
			handler.HandleMessage(ctx, v)
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "whatsapp:", err)
		os.Exit(1)
	}

	styleExtra := loadStyleExamples()
	handler = receptionist.New(cfg, appStore, aiClient, intentAI, plannerAI, collateAI, waClient, pbRepo, promptTpl, styleExtra, instructionsMD)

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	if pbClient.Enabled() {
		if err := pbClient.Ping(pingCtx); err != nil {
			fmt.Fprintln(os.Stderr, "WARNING: PocketBase ping failed:", err)
		} else {
			fmt.Println("PocketBase OK (", pbClient.BaseURL(), ")")
		}
	} else {
		fmt.Println("PocketBase disabled")
	}
	pingCancel()
	composioCfg, composioClient := calendar.ResolveComposioConfig(settingResolver)
	go appStore.RunCleanupLoop(ctx, store.DefaultCleanupConfig())

	var mailer composio.Mailer
	if composioCfg.GmailReady() && composioClient != nil {
		if es, err := composio.NewEmailService(composioClient, composioCfg); err == nil {
			mailer = es
		}
	}
	ops.SetWorkerEnv(ops.WorkerEnv{
		Store:  appStore,
		Cfg:    cfg,
		WA:     waClient,
		AI:     ai.AsAIFace(aiClient),
		Mailer: mailer,
	})
	go (&ops.AsyncWorker{
		Store:    appStore,
		Cfg:      cfg,
		WA:       waClient,
		Handlers: ops.DefaultJobHandlers(ops.WorkerEnv{Store: appStore, Cfg: cfg, WA: waClient, AI: ai.AsAIFace(aiClient), Mailer: mailer}),
		Interval: 30 * time.Second,
	}).Run(ctx)

	var api *httpapi.Server
	if httpAddr != "" {
		distDir := envOr("DASHBOARD_DIST", "dashboard/dist")
		api = httpapi.New(cfg, appStore, distDir, graphitiURL)
		api.SetWhatsAppClient(waClient)
		api.SetPromptMaterials(promptTpl, styleExtra, instructionsMD)
		api.SetPromptInvalidator(handler.InvalidatePromptCache)
		go func() {
			fmt.Println("HTTP API listening on", httpAddr)
			if err := api.Start(ctx, httpAddr); err != nil {
				fmt.Fprintln(os.Stderr, "http api:", err)
			}
		}()
	}

	if err := waClient.Start(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "start:", err)
		os.Exit(1)
	}

	fmt.Printf("%s running mode=%s provider=%s model=%s owner=%s groups=%v\n",
		cfg.BusinessName, cfg.Mode, aiClient.Name(), cfg.Model, cfg.OwnerNumber, cfg.ReplyToGroups)
	if waClient.WM.Store.ID == nil {
		fmt.Println("Waiting for QR scan...")
	} else {
		fmt.Println("Session linked — listening for messages")
		if id := waClient.WM.Store.ID; id != nil {
			fmt.Println("Linked account JID:", id.String(), "(set owner_number to this phone digits if testing self-chat)")
		}
	}

	if err := aiClient.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "WARNING: AI provider check failed:", err)
		fmt.Fprintln(os.Stderr, "WhatsApp will still connect but replies will fail until provider credentials are valid.")
	} else {
		fmt.Println("AI provider OK (provider:", aiClient.Name(), "model:", cfg.Model, ")")
	}

	go handler.RunNudgeLoop(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	cancel()
	fmt.Println("Shutting down...")
	if api != nil {
		shCtx, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
		_ = api.Shutdown(shCtx)
		cancel2()
	}
	waClient.Disconnect()
}
