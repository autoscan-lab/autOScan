# felituive

**felituive** is a local-first terminal UI for retrieval-augmented generation (RAG).

It lets you index local files, search them semantically, and chat with a local LLM — all from your terminal.  
By default, everything runs locally. More powerful backends (like MongoDB Atlas Vector Search) can be enabled optionally.

---

## Goals

- **Local-first**: works out of the box with no external services (besides Ollama)
- **Fast feedback**: simple workflows, responsive TUI
- **Extensible**: storage and retrieval backends are swappable
- **Product-quality UX**: feels like a real CLI app, not a demo

---

## High-level architecture

- **Language**: Go  
- **TUI**: Bubble Tea  
- **LLM & embeddings**: Ollama (local)  
- **Default storage**: SQLite + cosine similarity  
- **Optional backend**: MongoDB Atlas Vector Search  

The TUI is backend-agnostic. Storage and retrieval are abstracted behind a common interface.

---

## Core concepts

### Corpus
A *corpus* represents an indexed folder (or project).  
Each corpus contains many text *chunks* with embeddings.

### Chunk
A chunk is:
- a slice of text from a file
- its embedding
- metadata (path, chunk index, timestamps, etc.)

---

## Features (planned)

- Index a local folder into a corpus
- Semantic search over indexed content
- Chat with a local LLM using retrieved context
- Show sources for each answer
- Stream model responses in real time
- Switch between local and remote backends
- Fully keyboard-driven TUI

---

## Example workflow

```bash
# index a folder
felituive index --corpus notes --path ~/notes

# open chat TUI
felituive chat --corpus notes

# or open the main TUI
felituive tui
