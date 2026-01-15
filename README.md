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

## Usage (planned)

```bash
# index a folder
felituive index --corpus notes --path ~/notes

# open chat UI
felituive chat --corpus notes

# open the main TUI
felituive tui
