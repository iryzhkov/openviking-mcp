package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// JSON-RPC types

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string   `json:"jsonrpc"`
	ID      any      `json:"id,omitempty"`
	Result  any      `json:"result,omitempty"`
	Error   *rpcErr  `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// Tool definition

type toolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type prop struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     any    `json:"default,omitempty"`
}

type itemsProp struct {
	Type        string          `json:"type"`
	Description string          `json:"description"`
	Items       json.RawMessage `json:"items"`
}

func schema(props map[string]any, required []string) map[string]any {
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

// ov CLI wrapper

func ov(args []string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fullArgs := append(args, "-o", "json")
	cmd := exec.CommandContext(ctx, "ov", fullArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return "", fmt.Errorf("%s", out)
		}
		return "", err
	}
	return string(out), nil
}

// Tool definitions

func toolDefinitions() []toolDef {
	return []toolDef{
		// Read tools
		{
			Name: "ov_ls", Description: "List directory contents in OpenViking",
			InputSchema: schema(map[string]any{
				"uri":       prop{Type: "string", Description: "Viking URI to list", Default: "viking://"},
				"recursive": prop{Type: "boolean", Description: "List subdirectories recursively", Default: false},
			}, []string{"uri"}),
		},
		{
			Name: "ov_tree", Description: "Get directory tree in OpenViking",
			InputSchema: schema(map[string]any{
				"uri":         prop{Type: "string", Description: "Viking URI to get tree for"},
				"level_limit": prop{Type: "integer", Description: "Max depth level", Default: 3},
			}, []string{"uri"}),
		},
		{
			Name: "ov_read", Description: "Read full file content (L2) from OpenViking",
			InputSchema: schema(map[string]any{
				"uri": prop{Type: "string", Description: "Viking URI to read"},
			}, []string{"uri"}),
		},
		{
			Name: "ov_abstract", Description: "Read abstract (L0 summary) of a resource",
			InputSchema: schema(map[string]any{
				"uri": prop{Type: "string", Description: "Viking URI"},
			}, []string{"uri"}),
		},
		{
			Name: "ov_overview", Description: "Read overview (L1 summary) of a resource",
			InputSchema: schema(map[string]any{
				"uri": prop{Type: "string", Description: "Viking URI"},
			}, []string{"uri"}),
		},
		{
			Name: "ov_stat", Description: "Get resource metadata",
			InputSchema: schema(map[string]any{
				"uri": prop{Type: "string", Description: "Viking URI"},
			}, []string{"uri"}),
		},
		{
			Name: "ov_relations", Description: "List relations of a resource",
			InputSchema: schema(map[string]any{
				"uri": prop{Type: "string", Description: "Viking URI"},
			}, []string{"uri"}),
		},
		// Search tools
		{
			Name: "ov_find", Description: "Semantic search in OpenViking",
			InputSchema: schema(map[string]any{
				"query":      prop{Type: "string", Description: "Search query"},
				"uri":        prop{Type: "string", Description: "Target URI scope", Default: ""},
				"node_limit": prop{Type: "integer", Description: "Max results", Default: 10},
				"threshold":  prop{Type: "number", Description: "Score threshold"},
			}, []string{"query"}),
		},
		{
			Name: "ov_search", Description: "Context-aware search in OpenViking",
			InputSchema: schema(map[string]any{
				"query":      prop{Type: "string", Description: "Search query"},
				"uri":        prop{Type: "string", Description: "Target URI scope", Default: ""},
				"node_limit": prop{Type: "integer", Description: "Max results", Default: 10},
				"session_id": prop{Type: "string", Description: "Session ID for context"},
				"threshold":  prop{Type: "number", Description: "Score threshold"},
			}, []string{"query"}),
		},
		{
			Name: "ov_grep", Description: "Pattern search in OpenViking content",
			InputSchema: schema(map[string]any{
				"pattern":     prop{Type: "string", Description: "Search pattern"},
				"uri":         prop{Type: "string", Description: "Target URI", Default: "viking://"},
				"ignore_case": prop{Type: "boolean", Description: "Case insensitive", Default: false},
				"node_limit":  prop{Type: "integer", Description: "Max results", Default: 256},
			}, []string{"pattern"}),
		},
		{
			Name: "ov_glob", Description: "Glob pattern search for files in OpenViking",
			InputSchema: schema(map[string]any{
				"pattern":    prop{Type: "string", Description: "Glob pattern"},
				"uri":        prop{Type: "string", Description: "Search root URI", Default: "viking://"},
				"node_limit": prop{Type: "integer", Description: "Max results", Default: 256},
			}, []string{"pattern"}),
		},
		// Write tools
		{
			Name: "ov_add_resource", Description: "Add a resource (file, URL, or directory) into OpenViking",
			InputSchema: schema(map[string]any{
				"path":        prop{Type: "string", Description: "Local path or URL to import"},
				"to":          prop{Type: "string", Description: "Exact target URI (must not exist yet)"},
				"parent":      prop{Type: "string", Description: "Target parent URI (must exist, must be a directory)"},
				"reason":      prop{Type: "string", Description: "Reason for import", Default: ""},
				"instruction": prop{Type: "string", Description: "Additional instruction", Default: ""},
				"wait":        prop{Type: "boolean", Description: "Wait until processing is complete", Default: false},
			}, []string{"path"}),
		},
		{
			Name: "ov_add_skill", Description: "Add a skill into OpenViking",
			InputSchema: schema(map[string]any{
				"data": prop{Type: "string", Description: "Skill directory, SKILL.md path, or raw content"},
				"wait": prop{Type: "boolean", Description: "Wait until processing is complete", Default: false},
			}, []string{"data"}),
		},
		{
			Name: "ov_add_memory", Description: "Add a memory to OpenViking (creates session, adds messages, commits)",
			InputSchema: schema(map[string]any{
				"content": prop{Type: "string", Description: "Content to memorize. Plain string or JSON message(s)"},
			}, []string{"content"}),
		},
		{
			Name: "ov_mkdir", Description: "Create a directory in OpenViking",
			InputSchema: schema(map[string]any{
				"uri": prop{Type: "string", Description: "Directory URI to create"},
			}, []string{"uri"}),
		},
		{
			Name: "ov_rm", Description: "Remove a resource from OpenViking",
			InputSchema: schema(map[string]any{
				"uri":       prop{Type: "string", Description: "Viking URI to remove"},
				"recursive": prop{Type: "boolean", Description: "Remove recursively", Default: false},
			}, []string{"uri"}),
		},
		{
			Name: "ov_mv", Description: "Move or rename a resource in OpenViking",
			InputSchema: schema(map[string]any{
				"from_uri": prop{Type: "string", Description: "Source URI"},
				"to_uri":   prop{Type: "string", Description: "Target URI"},
			}, []string{"from_uri", "to_uri"}),
		},
		{
			Name: "ov_link", Description: "Create relation links between resources",
			InputSchema: schema(map[string]any{
				"from_uri": prop{Type: "string", Description: "Source URI"},
				"to_uris":  itemsProp{Type: "array", Description: "Target URIs", Items: json.RawMessage(`{"type":"string"}`)},
				"reason":   prop{Type: "string", Description: "Reason for linking", Default: ""},
			}, []string{"from_uri", "to_uris"}),
		},
		{
			Name: "ov_unlink", Description: "Remove a relation link between resources",
			InputSchema: schema(map[string]any{
				"from_uri": prop{Type: "string", Description: "Source URI"},
				"to_uri":   prop{Type: "string", Description: "Target URI to unlink"},
			}, []string{"from_uri", "to_uri"}),
		},
	}
}

// Tool dispatch

func callTool(params callToolParams) (string, bool) {
	a := params.Arguments
	str := func(key string) string {
		if v, ok := a[key].(string); ok {
			return v
		}
		return ""
	}
	num := func(key string, def int) int {
		if v, ok := a[key].(float64); ok {
			return int(v)
		}
		return def
	}
	boolean := func(key string) bool {
		if v, ok := a[key].(bool); ok {
			return v
		}
		return false
	}

	var args []string
	timeout := 60 * time.Second

	switch params.Name {
	case "ov_ls":
		args = []string{"ls", str("uri")}
		if boolean("recursive") {
			args = append(args, "-r")
		}

	case "ov_tree":
		args = []string{"tree", str("uri"), "-L", strconv.Itoa(num("level_limit", 3))}

	case "ov_read":
		args = []string{"read", str("uri")}

	case "ov_abstract":
		args = []string{"abstract", str("uri")}

	case "ov_overview":
		args = []string{"overview", str("uri")}

	case "ov_stat":
		args = []string{"stat", str("uri")}

	case "ov_relations":
		args = []string{"relations", str("uri")}

	case "ov_find":
		args = []string{"find", str("query"), "-n", strconv.Itoa(num("node_limit", 10))}
		if u := str("uri"); u != "" {
			args = append(args, "-u", u)
		}
		if v, ok := a["threshold"].(float64); ok {
			args = append(args, "-t", strconv.FormatFloat(v, 'f', -1, 64))
		}

	case "ov_search":
		args = []string{"search", str("query"), "-n", strconv.Itoa(num("node_limit", 10))}
		if u := str("uri"); u != "" {
			args = append(args, "-u", u)
		}
		if s := str("session_id"); s != "" {
			args = append(args, "--session-id", s)
		}
		if v, ok := a["threshold"].(float64); ok {
			args = append(args, "-t", strconv.FormatFloat(v, 'f', -1, 64))
		}

	case "ov_grep":
		args = []string{"grep", str("pattern"), "-u", str("uri"), "-n", strconv.Itoa(num("node_limit", 256))}
		if boolean("ignore_case") {
			args = append(args, "-i")
		}

	case "ov_glob":
		args = []string{"glob", str("pattern"), "-u", str("uri"), "-n", strconv.Itoa(num("node_limit", 256))}

	case "ov_add_resource":
		args = []string{"add-resource", str("path")}
		if v := str("to"); v != "" {
			args = append(args, "--to", v)
		}
		if v := str("parent"); v != "" {
			args = append(args, "--parent", v)
		}
		if v := str("reason"); v != "" {
			args = append(args, "--reason", v)
		}
		if v := str("instruction"); v != "" {
			args = append(args, "--instruction", v)
		}
		if boolean("wait") {
			args = append(args, "--wait", "--timeout", "120")
			timeout = 180 * time.Second
		}

	case "ov_add_skill":
		args = []string{"add-skill", str("data")}
		if boolean("wait") {
			args = append(args, "--wait")
		}

	case "ov_add_memory":
		args = []string{"add-memory", str("content")}
		timeout = 120 * time.Second

	case "ov_mkdir":
		args = []string{"mkdir", str("uri")}

	case "ov_rm":
		args = []string{"rm", str("uri")}
		if boolean("recursive") {
			args = append(args, "-r")
		}

	case "ov_mv":
		args = []string{"mv", str("from_uri"), str("to_uri")}

	case "ov_link":
		args = []string{"link", str("from_uri")}
		if uris, ok := a["to_uris"].([]any); ok {
			for _, u := range uris {
				if s, ok := u.(string); ok {
					args = append(args, s)
				}
			}
		}
		if v := str("reason"); v != "" {
			args = append(args, "--reason", v)
		}

	case "ov_unlink":
		args = []string{"unlink", str("from_uri"), str("to_uri")}

	default:
		return fmt.Sprintf("unknown tool: %s", params.Name), true
	}

	result, err := ov(args, timeout)
	if err != nil {
		return err.Error(), true
	}
	return result, false
}

// Server

func handleRequest(req request) response {
	switch req.Method {
	case "initialize":
		return response{
			JSONRPC: "2.0", ID: req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":   map[string]any{"tools": map[string]any{}},
				"serverInfo":     map[string]any{"name": "openviking", "version": "1.0.0"},
			},
		}

	case "notifications/initialized":
		return response{}

	case "tools/list":
		return response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": toolDefinitions()}}

	case "tools/call":
		var params callToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return response{JSONRPC: "2.0", ID: req.ID, Error: &rpcErr{Code: -32602, Message: "invalid params"}}
		}
		result, isErr := callTool(params)
		return response{
			JSONRPC: "2.0", ID: req.ID,
			Result: map[string]any{
				"content": []textContent{{Type: "text", Text: result}},
				"isError": isErr,
			},
		}

	default:
		return response{JSONRPC: "2.0", ID: req.ID, Error: &rpcErr{Code: -32601, Message: "method not found"}}
	}
}

func writeResponse(w io.Writer, resp response) {
	if resp.JSONRPC == "" {
		return
	}
	data, _ := json.Marshal(resp)
	fmt.Fprintf(w, "%s\n", data)
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			return
		}

		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			writeResponse(writer, response{JSONRPC: "2.0", Error: &rpcErr{Code: -32700, Message: "parse error"}})
			continue
		}

		resp := handleRequest(req)
		writeResponse(writer, resp)
	}
}
