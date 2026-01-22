<h1 align="center">autOScan</h1>

<p align="center">
  <strong>TUI tool for grading C lab submissions</strong>
</p>

<p align="center">
  <a href="#"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat&logo=go&logoColor=white" /></a>
  <a href="#"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-000000?style=flat" /></a>
  <a href="#"><img src="https://img.shields.io/badge/compiler-gcc-A42E2B?style=flat" /></a>
  <a href="#"><img src="https://img.shields.io/badge/license-MIT-24292e?style=flat" /></a>
</p>

---

## Installation

Download the binary from [Releases](https://github.com/Feli05/autOScan/releases), then:

```bash
chmod +x autoscan-darwin-arm64
./autoscan-darwin-arm64
```

On first run, it auto-installs to `~/.local/bin/autoscan` and prompts you to add to PATH if needed.

### Build from Source

```bash
git clone https://github.com/Feli05/autOScan.git
cd autOScan
make install
```

**Requires:** Go 1.22+, gcc

---

## Usage

```bash
autoscan
```

Use arrow keys to navigate, Enter to select. Export to Markdown, JSON, or CSV.

---

## Configuration

On first run, configs are created at `~/.config/autoscan/`:

```
~/.config/autoscan/
├── policies/        # Policy YAML files
│   └── example.yaml
└── banned.txt       # Global banned functions
```

**Policy example:**

```yaml
name: "Lab 03"
compile:
  flags: ["-Wall", "-Wextra", "-lpthread"]
  output: "lab03"
required_files: [S0.c, S1.c]
```

**Banned Functions:** Edit `~/.config/autoscan/banned.txt` (one function per line, `#` for comments).

---

## License

MIT
