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

## Features

- Batch compile and grade C submissions
- Detect banned function calls (e.g., `printf`, `fprintf`)
- Create and manage grading policies
- Filter results by status (pass/fail/banned)
- Export results to JSON or CSV
- Interactive folder browser for selecting submission directories

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

### Navigation

- **↑/↓** - Navigate lists
- **Enter** - Select/confirm
- **Esc** - Go back
- **Tab** - Switch tabs (in detail view)

### Main Menu

1. **Run Grader** - Select a policy and grade submissions
2. **Manage Policies** - Create, edit, or delete grading policies
3. **Settings** - Configure display options
4. **Uninstall** - Remove autoscan and configs

### Grading Results

- **[OK]** - Compiled successfully, no banned calls
- **[!]** - Has banned function calls
- **[X]** - Compilation failed
- **CHECK** - Requires manual testing
- **2** - Automatic fail grade (compile error or banned calls)

---

## Configuration

On first run, configs are created at `~/.config/autoscan/`:

```
~/.config/autoscan/
├── policies/        # Policy YAML files
│   └── example.yaml
├── libraries/       # Bundled library files (.c/.h)
├── banned.yaml      # Global banned functions
└── settings.yaml    # User preferences
```

### Policy Example

```yaml
name: "Lab 03 - Processes"
compile:
  gcc: "gcc"
  flags: ["-Wall", "-Wextra", "-lpthread"]
  output: "lab03"
required_files: [S0.c, S1.c]
library_files:
  - /path/to/lib/utils.c
  - /path/to/lib/helper.h
```

### Library Files

Library files are additional source files that get compiled with each student submission. This is useful for shared header files or instructor-provided utilities.

When you add a library file, it is **copied** to `~/.config/autoscan/libraries/` so it stays bundled with autoscan. You can run autoscan from anywhere and the library files will be available.

Add library files via the policy editor (Manage Policies → select policy → Library Files section) using the file browser.

### Banned Functions

Edit via the TUI (Manage Policies → Edit Banned Functions) or directly in `~/.config/autoscan/banned.yaml`:

```yaml
banned:
  - printf
  - fprintf
  - puts
```

### Settings

- **Short Names** - Truncate folder names at first underscore
- **Keep Binaries** - Preserve compiled binaries after grading

---

## Export

Export grading results to:

- **JSON** - Structured data for further processing
- **CSV** - Spreadsheet compatible format

Files are saved to the current working directory.

---

## License

MIT
