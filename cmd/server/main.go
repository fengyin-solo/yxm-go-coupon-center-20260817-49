package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"couponcenter/internal/app"
	"couponcenter/internal/config"
	"couponcenter/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.NewLevel(logger.ParseLevel(os.Getenv("LOG_LEVEL")))

	application, err := app.New(cfg, log)
	if err != nil {
		log.Errorf("应用初始化失败: %v", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           application.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Infof("服务已启动，监听 %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("服务启动失败: %v", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infof("正在关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("服务关闭失败: %v", err)
	}
	log.Infof("服务已关闭")
}
