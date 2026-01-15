<h1 align="center">felituive</h1>

<p align="center">
  <strong>Local-first retrieval-augmented generation in your terminal</strong>
</p>

<p align="center">
  A terminal UI for indexing, searching, and chatting with your files using a local LLM.
</p>

<p align="center">
  <a href="#"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat&logo=go&logoColor=white" /></a>
  <a href="#"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-000000?style=flat" /></a>
  <a href="#"><img src="https://img.shields.io/badge/LLM-Ollama-24292e?style=flat" /></a>
  <a href="#"><img src="https://img.shields.io/badge/license-MIT-24292e?style=flat" /></a>
</p>

<p align="center">
  <em>Local by default. Extensible by design.</em>
</p>

---

## Installation

### From source

```bash
git clone https://github.com/felipetrejos/felituive.git
cd felituive
make install
```

### Prerequisites

[Ollama](https://ollama.ai/) running locally:

```bash
ollama pull llama3.2
ollama pull nomic-embed-text
```

---

## Usage

Simply run the command to launch the interactive TUI:

```bash
felituive
```

Navigate using your keyboard:
- `1` — Index a folder
- `2` — Semantic search
- `3` — Chat with your files
- `q` — Quit

All features are accessible from within the TUI — no subcommands needed.

---

## Overview

**felituive** is a local-first terminal UI for retrieval-augmented generation (RAG).

It lets you index folders, perform semantic search, and chat with a local LLM — all from the terminal.  
No external services are required by default. Remote backends can be enabled optionally for larger workloads.

---

## Why felituive

- **Local-first** — works out of the box with no cloud dependencies
- **Fast feedback** — designed for tight iteration loops
- **Extensible** — pluggable storage and retrieval backends
- **Terminal-native** — fully keyboard-driven TUI
- **Product-oriented** — focused on UX, not just demos

---

## Architecture

- **Language**: Go  
- **TUI**: Bubble Tea  
- **LLM & embeddings**: Ollama  
- **Default backend**: SQLite + cosine similarity  
- **Optional backend**: MongoDB Atlas Vector Search  

The TUI is backend-agnostic. Storage and retrieval are implemented behind a shared interface.

```
felituive/
├── cmd/felituive/         # Entry point
├── internal/
│   ├── app/               # Application bootstrap
│   ├── domain/            # Core business types
│   ├── ports/             # Interface definitions
│   ├── adapters/          # Backend implementations
│   ├── services/          # Use cases & orchestration
│   └── tui/               # Bubble Tea UI layer
├── pkg/                   # Shared utilities
└── configs/               # Default configuration
```

---

## Core concepts

### Corpus
A *corpus* represents an indexed folder or project.

### Chunk
A chunk is a slice of text extracted from a file, along with:
- its embedding
- file path and position
- metadata (timestamps, type, etc.)

---

## Features

- Index local folders into named corpora
- Semantic search over indexed content
- Chat with a local LLM using retrieved context
- Source attribution for each response
- Streaming model output
- Multiple backends (local and remote)
- Fully keyboard-driven interface

---

## Development

```bash
# Build
make build

# Run
make run

# Test
make test

# Build for all platforms
make build-all
```

---

## Configuration

Configuration is stored in `~/.felituive/config.yaml`. See [configs/default.yaml](configs/default.yaml) for available options.

---

## Roadmap

- [ ] Core indexing pipeline
- [ ] SQLite vector storage
- [ ] Semantic search
- [ ] RAG chat with streaming
- [ ] MongoDB Atlas backend
- [ ] Watch mode (auto-reindex)
- [ ] Session history
- [ ] Multiple embedding models

---

## License

MIT
