<h1 align="center">autOScan</h1>

<p align="center">
  <strong>Automated C lab submission grader for instructors</strong>
</p>

<p align="center">
  <a href="#"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat&logo=go&logoColor=white" /></a>
  <a href="#"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-000000?style=flat" /></a>
  <a href="#"><img src="https://img.shields.io/badge/compiler-gcc-A42E2B?style=flat" /></a>
  <a href="#"><img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey?style=flat" /></a>
  <a href="#"><img src="https://img.shields.io/badge/license-MIT-24292e?style=flat" /></a>
</p>

---

## Installation

```bash
# Download the binary, then:
./autoscan install
```

Installs to `~/.local/bin/autoscan`. If `~/.local/bin` isn't in your PATH:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

To uninstall:
```bash
autoscan uninstall
```

### Build from Source

```bash
git clone https://github.com/felipetrejos/autoscan.git
cd autoscan
make install
```

**Requires:** Go 1.22+, gcc

---

## Platform Support

| Platform | Status |
|----------|--------|
| macOS    | ✅ Supported |
| Linux    | ✅ Supported |
| Windows  | ❌ Not yet   |

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
