package tools

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func structuredToolResult(out any) (*mcp.CallToolResult, any, error) {
	result, err := structuredToolResultWithError(out, false)
	if err != nil {
		return nil, nil, err
	}
	return result, out, nil
}

func structuredToolResultWithError(out any, isError bool) (*mcp.CallToolResult, error) {
	resultJSON, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(resultJSON)}},
		StructuredContent: out,
		IsError:           isError,
	}, nil
}
