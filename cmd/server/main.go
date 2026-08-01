// Command server 启动 slim-agent HTTP 工作台服务。
// 装配 harness 存储、workbench 应用层、鉴权中间件与静态文件嵌入。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"slim-agent/internal/harness"
	"slim-agent/internal/providers"
	"slim-agent/internal/skills"
	"slim-agent/internal/skills/literature_search"
	"slim-agent/internal/workbench"
	webdist "slim-agent" // 项目根包（webdist.go，嵌入 web/dist）

	_ "modernc.org/sqlite"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("slim-agent: %v", err)
	}
}

func run() error {
	// ---- 配置装配：flag > env > 默认 ----
	addr := flag.String("addr", defaultAddr(), "listen address")
	dataDir := flag.String("data-dir", ".harness-data", "data directory")
	corsFlag := flag.String("cors-origins", "", "comma-separated CORS origin whitelist (empty disables)")
	bypass := flag.Bool("auth-bypass", false, "development mode: skip auth")
	flag.Parse()

	if port := os.Getenv("HARNESS_API_PORT"); port != "" {
		*addr = "127.0.0.1:" + port
	}
	if dir := os.Getenv("HARNESS_DATA_DIR"); dir != "" {
		*dataDir = dir
	}

	// ---- 存储装配 ----
	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// workbench（独立 SQLite 文件，避免与 harness 存储竞争同一连接）
	wbPath := filepath.Join(*dataDir, "workbench.sqlite3")
	wbDB, err := sql.Open("sqlite", wbPath)
	if err != nil {
		return fmt.Errorf("open workbench db: %w", err)
	}
	defer wbDB.Close()
	wbStore := workbench.NewWorkbenchStore(wbDB)
	ctx := context.Background()
	if err := wbStore.InitAll(ctx); err != nil {
		return fmt.Errorf("init workbench: %w", err)
	}
	firstAccount, err := wbStore.EnsureDefaultAccount(ctx)
	if err != nil {
		return fmt.Errorf("ensure default account: %w", err)
	}
	if firstAccount == nil {
		log.Println("no accounts found — open the web UI to register the first account")
	}

	// harness（领域无关核心存储）
	harnessStore := harness.NewHarnessStore(*dataDir)
	if err := harnessStore.Initialize(); err != nil {
		return fmt.Errorf("init harness store: %w", err)
	}
	defer harnessStore.Close()

	// ---- 运行时装配 ----
	// Provider：优先使用环境变量 HARNESS_PROVIDER_CONFIG 覆盖 preset 字段。
	providerConfig := providers.PresetConfig(providers.PresetDeepSeek)
	if cfg := os.Getenv("HARNESS_PROVIDER_CONFIG"); cfg != "" {
		var overrides map[string]string
		if err := json.Unmarshal([]byte(cfg), &overrides); err == nil {
			if v, ok := overrides["provider_id"]; ok {
				providerConfig.ProviderID = v
			}
			if v, ok := overrides["base_url"]; ok {
				providerConfig.BaseURL = v
			}
			if v, ok := overrides["model_id"]; ok {
				providerConfig.ModelID = v
			}
			if v, ok := overrides["api_key_env"]; ok {
				providerConfig.APIKeyEnv = v
			}
		}
	}
	provider, pErr := providers.NewOpenAICompatibleProvider(providerConfig)
	if pErr != nil {
		log.Printf("WARN: provider init failed (%v); runtime will be disabled", pErr)
	}

	// 验证器注册表：内建验证器 + 领域验证器
	valReg := harness.NewValidatorRegistry()
	_ = valReg.Register(harness.NewArtifactExistsValidator())
	_ = valReg.Register(harness.NewJSONSchemaValidator())
	_ = valReg.Register(harness.NewReferenceIntegrityValidator())
	_ = valReg.Register(harness.NewBudgetValidator())

	// 工具执行器（默认策略：高危需审批，空白名单默认拒绝）
	toolGov := harness.NewToolGovernor(harness.DefaultPolicy())

	// Skill 注册表
	skillReg := skills.NewRegistry()
	_ = skillReg.Register(literature_search.NewSkill())

	// 事件中心（外部实例，同时供 Server 和 Runtime 使用）
	hub := workbench.NewEventHub()

	// 运行时执行引擎
	var runtime *harness.HarnessRuntime
	if provider != nil {
		runtime, pErr = harness.NewHarnessRuntime(harnessStore, harness.RuntimeOptions{
			Provider:   provider,
			ProviderID: providerConfig.ProviderID,
			ModelID:    providerConfig.ModelID,
			Tools:      toolGov,
			Validators: valReg,
			OnEvent:    func(ev *harness.Event) { hub.Broadcast(ev) },
		})
		if pErr != nil {
			log.Printf("WARN: runtime init failed (%v); runtime will be disabled", pErr)
			runtime = nil
		}
	}

	// ---- HTTP 服务 ----
	var corsOrigins []string
	if *corsFlag != "" {
		for _, o := range strings.Split(*corsFlag, ",") {
			if o = strings.TrimSpace(o); o != "" {
				corsOrigins = append(corsOrigins, o)
			}
		}
	}
	server := workbench.NewServer(wbStore, harnessStore, workbench.ServerOptions{
		CORSOrigins: corsOrigins,
		AuthBypass:  *bypass,
		Skills:      skillReg,
		Runtime:     runtime,
		EventHub:    hub,
	})

	fsys, err := fs.Sub(webdist.FS, "web/dist")
	if err != nil {
		return fmt.Errorf("sub web/dist: %w", err)
	}
	fileServer := http.FileServer(http.FS(fsys))
	apiHandler := server.Handler()

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           spaHandler(apiHandler, fileServer, fsys),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// ---- 优雅关闭 ----
	errCh := make(chan error, 1)
	go func() {
		log.Printf("slim-agent listening on http://%s (data: %s)", *addr, *dataDir)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case sig := <-stop:
		log.Printf("received %s, shutting down...", sig)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
	}
	return nil
}

// defaultAddr 返回默认监听地址。
func defaultAddr() string {
	if port := os.Getenv("HARNESS_API_PORT"); port != "" {
		return "127.0.0.1:" + port
	}
	return "127.0.0.1:8000"
}

// spaHandler 路由 API 到后端，其余路径服务静态文件；静态文件缺失时 fallback index.html。
func spaHandler(api http.Handler, files http.Handler, fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
			api.ServeHTTP(w, r)
			return
		}
		clean := strings.TrimPrefix(filepath.Clean(r.URL.Path), string(filepath.Separator))
		if clean == "." {
			clean = "index.html"
		}
		if _, err := fs.Stat(fsys, clean); err != nil {
			// SPA fallback：非资源路径回退 index.html
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			files.ServeHTTP(w, r2)
			return
		}
		files.ServeHTTP(w, r)
	})
}
