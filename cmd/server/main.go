package main

import (
	"context"
	"flag"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/rickliujh/trading-chat-aggr/pkg/api/v1/apiv1connect"
	"github.com/rickliujh/trading-chat-aggr/pkg/server"
	"github.com/rickliujh/trading-chat-aggr/pkg/sql"
	"github.com/rickliujh/trading-chat-aggr/pkg/utils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	address     string
	name        = "server"
	environment = "development"
	dburi       = "postgres://username:password@localhost:5432/database_name"

	// set at build time
	version = "v0.0.1-default"
)

func main() {
	flag.StringVar(&address, "address", ":8080", "Server address (host:port)")
	flag.StringVar(&name, "name", name, "Server name (default: server)")
	flag.StringVar(&environment, "environment", environment, "Server environment (default: development)")
	flag.StringVar(&dburi, "dburi", dburi, "Server pgxdb uri(default: localhost)")
	flag.Parse()

	logger := utils.NewLogger(4)

	// create server
	logger.Info("creating server...")
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dburi)
	if err != nil {
		logger.Error(err, "unable to connect to db")
		return
	}
	defer conn.Close(ctx)
	queries := sql.New(conn)

	done := make(chan struct{})

	s, err := server.NewService(*logger, queries, done)
	if err != nil {
		logger.Error(err, "error while creating server")
		return
	}

	mux := http.NewServeMux()
	path, handler := apiv1connect.NewAggrHandler(s)
	mux.Handle(path, handler)

	// run server
	logger.Info("starting server...")
	server := http.Server{
		Addr:    address,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	server.RegisterOnShutdown(func() {
		close(done)
	})

	if err := server.ListenAndServe(); err != nil {
		logger.Error(err, "error while running server")
		return
	}
}
