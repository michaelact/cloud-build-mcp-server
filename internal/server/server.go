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
	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio or http)")
	flag.StringVar(
		&transport,
		"transport",
		"stdio",
		"Transport type (stdio or http)",
	)
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
			if err := server.ServeStdio(s); err != nil {
				fmt.Printf("Server error: %v\n", err)
			}

		case "http":
			sseServer := server.NewSSEServer(s)
			if err := sseServer.Start(":8080"); err != nil {
				fmt.Printf("Server error: %v\n", err)
			}

		default:
			log.Fatalf(
				"Invalid transport type: %s. Must be 'stdio' or 'http'",
				transport,
			)
	}
}
