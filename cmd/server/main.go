package main

import (
	"context"
	"flag"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/rickliujh/kickstart-gogrpc/pkg/api/v1/apiv1connect"
	"github.com/rickliujh/kickstart-gogrpc/pkg/server"
	"github.com/rickliujh/kickstart-gogrpc/pkg/sql"
	"github.com/rickliujh/kickstart-gogrpc/pkg/utils"
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

	logger := utils.NewLogger(0)

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

	s, err := server.NewServer(name, version, environment, queries)
	if err != nil {
		logger.Error(err, "error while creating server")
		return
	}

	mux := http.NewServeMux()
	path, handler := apiv1connect.NewServiceHandler(s)
	mux.Handle(path, handler)

	// run server
	logger.Info("starting server...", "server_name", s.String())
	if err := http.ListenAndServe(
		address,
		h2c.NewHandler(mux, &http2.Server{}),
	); err != nil {
		logger.Error(err, "error while running server")
		return
	}

	logger.Info("done")
}
