package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	cloudbuildpb "cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"cloud.google.com/go/storage"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const defaultTailLines = 100

func getCloudBuildLogs() server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("get_cloud_build_logs",
			mcp.WithDescription("Get logs for a specific Google Cloud Build job"),
			mcp.WithString("project_id",
				mcp.Required(),
				mcp.Description("GCP project ID"),
			),
			mcp.WithString("build_id",
				mcp.Required(),
				mcp.Description("Cloud Build job ID"),
			),
			mcp.WithNumber("tail_lines",
				mcp.Description("Number of log lines to return from the end (default: 100, use 0 for all logs)"),
			),
		),
		Handler: getCloudBuildLogsHandler,
	}
}

func getCloudBuildLogsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := request.RequireString("project_id")
	if err != nil {
		return mcp.NewToolResultError("Missing project_id: " + err.Error()), nil
	}
	buildID, err := request.RequireString("build_id")
	if err != nil {
		return mcp.NewToolResultError("Missing build_id: " + err.Error()), nil
	}

	// Get tail_lines parameter, default to 100
	tailLines := request.GetInt("tail_lines", defaultTailLines)

	// Create Cloud Build client to get build info
	buildClient, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return mcp.NewToolResultError("Failed to create Cloud Build client: " + err.Error()), nil
	}
	defer buildClient.Close()

	// Get the build to retrieve log information
	build, err := buildClient.GetBuild(ctx, &cloudbuildpb.GetBuildRequest{
		ProjectId: projectID,
		Id:        buildID,
	})
	if err != nil {
		return mcp.NewToolResultError("Error getting build: " + err.Error()), nil
	}

	// Try to fetch logs from Cloud Storage
	logContent, err := fetchLogsFromGCS(ctx, build.LogsBucket, build.Id, tailLines)
	if err != nil {
		// If we can't fetch logs, return the log URL instead
		result := map[string]interface{}{
			"build_id":   build.Id,
			"status":     build.Status.String(),
			"log_url":    build.LogUrl,
			"error":      fmt.Sprintf("Could not fetch log content: %v", err),
		}
		resultBytes, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(resultBytes)), nil
	}

	result := map[string]interface{}{
		"build_id":    build.Id,
		"status":      build.Status.String(),
		"log_url":     build.LogUrl,
		"log_content": logContent,
	}
	resultBytes, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultBytes)), nil
}

// fetchLogsFromGCS fetches the build logs from Google Cloud Storage
// If tailLines > 0, only the last tailLines lines are returned
func fetchLogsFromGCS(ctx context.Context, logsBucket string, buildID string, tailLines int) (string, error) {
	if logsBucket == "" {
		return "", fmt.Errorf("logs bucket not available")
	}

	// Remove gs:// prefix if present
	bucket := strings.TrimPrefix(logsBucket, "gs://")

	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create storage client: %w", err)
	}
	defer client.Close()

	// Cloud Build logs are stored as log-{build-id}.txt
	objectName := fmt.Sprintf("log-%s.txt", buildID)
	reader, err := client.Bucket(bucket).Object(objectName).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read log object: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read log content: %w", err)
	}

	logText := string(content)

	// If tailLines is 0 or negative, return all logs
	if tailLines <= 0 {
		return logText, nil
	}

	// Split into lines and return only the last tailLines
	lines := strings.Split(logText, "\n")
	if len(lines) <= tailLines {
		return logText, nil
	}

	// Get the last tailLines lines
	tailedLines := lines[len(lines)-tailLines:]
	return strings.Join(tailedLines, "\n"), nil
}
