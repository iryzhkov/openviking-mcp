# openviking-mcp

A lightweight Go MCP server that exposes [OpenViking](https://openviking.io) knowledge base operations as tools for AI assistants via the [Model Context Protocol](https://modelcontextprotocol.io).

This is a thin wrapper around the `ov` CLI — it translates MCP tool calls into `ov` commands and returns the results.

## Prerequisites

- The `ov` CLI must be installed and available in your `PATH`
- An OpenViking server must be running and configured

## Installation

```bash
go build -o openviking-mcp .
```

## Usage

```bash
openviking-mcp
```

The server communicates over stdio using JSON-RPC (MCP protocol).

## Available Tools

### Read

| Tool | Description |
|---|---|
| `ov_ls` | List directory contents |
| `ov_tree` | Get directory tree |
| `ov_read` | Read full file content (L2) |
| `ov_abstract` | Read abstract (L0 summary) |
| `ov_overview` | Read overview (L1 summary) |
| `ov_stat` | Get resource metadata |
| `ov_relations` | List relations of a resource |

### Search

| Tool | Description |
|---|---|
| `ov_find` | Semantic search |
| `ov_search` | Context-aware search |
| `ov_grep` | Pattern/text search |
| `ov_glob` | Glob pattern file search |

### Write

| Tool | Description |
|---|---|
| `ov_add_resource` | Import a file, URL, or directory |
| `ov_add_skill` | Store a reusable skill/workflow |
| `ov_add_memory` | Persist a memory |
| `ov_mkdir` | Create a directory |
| `ov_rm` | Remove a resource |
| `ov_mv` | Move/rename a resource |
| `ov_link` | Create relation links |
| `ov_unlink` | Remove a relation link |

## Claude Desktop Integration

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "openviking": {
      "command": "/path/to/openviking-mcp"
    }
  }
}
```

## Claude Code Integration

Add to `.claude.json` or `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "openviking": {
      "type": "stdio",
      "command": "/path/to/openviking-mcp",
      "args": []
    }
  }
}
```

## License

MIT
