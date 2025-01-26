package main

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/rickliujh/trading-chat-aggr/pkg/api/v1/apiv1connect"
	"github.com/rickliujh/trading-chat-aggr/pkg/server"
	"github.com/rickliujh/trading-chat-aggr/pkg/sql"
	"github.com/rickliujh/trading-chat-aggr/pkg/utils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	conf, err := loadConfig()
	if err != nil {
		panic(err)
	}

	logger := utils.NewLogger(conf.LogLevel)

	logger.Info("starting server...")
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, conf.DBURI)
	if err != nil {
		logger.Error(err, "unable to connect to db")
		return
	}
	defer conn.Close(ctx)
	queries := sql.New(conn)

	done := make(chan struct{})

	s, err := server.NewService(*logger, queries, conf.Symbols, done)
	if err != nil {
		logger.Error(err, "error while creating server")
		return
	}

	mux := http.NewServeMux()
	path, handler := apiv1connect.NewAggrHandler(s)
	mux.Handle(path, handler)

	logger.Info("running...")
	server := http.Server{
		Addr:    conf.Addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	server.RegisterOnShutdown(func() {
		logger.Info("shutting down server...")
		close(done)
	})

	if err := server.ListenAndServe(); err != nil {
		logger.Error(err, "error while running server")
		return
	}
}
