package server

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"

	"github.com/jerop/cloud-build-mcp-server/internal/tools"
	"github.com/mark3labs/mcp-go/server"
)

// StartServer initializes and starts the MCP server.
func StartServer() {
	var transport string
	var address string

	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio or http)")
	flag.StringVar(&transport, "transport", "stdio", "Transport type (stdio or http)")
	flag.StringVar(&address, "address", "127.0.0.1:8080", "Address to listen on")

	flag.Parse()

	s := server.NewMCPServer(
		"Cloud Build MCP Server",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)
	slog.Info("Adding tools and resources to the server.")
	ctx := context.Background()
	tools.Add(ctx, s)

	switch transport {
		case "stdio":
			slog.Info("Starting server with stdio transport")
			if err := server.ServeStdio(s); err != nil {
				fmt.Printf("Server error: %v\n", err)
			}

		case "http":
			slog.Info("Starting server with HTTP transport", "address", address, "path", "/sse")
			sseServer := server.NewSSEServer(s)
			if err := sseServer.Start(address); err != nil {
				fmt.Printf("Server error: %v\n", err)
			}

		default:
			log.Fatalf(
				"Invalid transport type: %s. Must be 'stdio' or 'http'",
				transport,
			)
	}
}
