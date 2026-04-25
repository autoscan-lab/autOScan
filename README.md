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

<p align="center">
  <img src="screenshots/autoscan.png" alt="autOScan Screenshot" />
</p>

<p align="center">
  <a href="https://autoscan-web.vercel.app">autoscan-web.vercel.app</a>
</p>

---

## What It Does

`autOScan` helps you grade C submissions in bulk with policy-driven compile/run checks.

- Batch compile and classify submissions
- Detect banned function calls
- Run submissions with manual args/stdin or policy test cases
- Run multi-process labs with streaming output
- Compare output against expected files (diff view)
- Similarity detection across submissions
- AI-pattern detection against a configurable dictionary

---

## Installation

**Supported platforms:** macOS (arm64), Linux (amd64)

Download binaries from [Releases](https://github.com/autoscan-lab/autOScan/releases).

**macOS:**
```bash
chmod +x autoscan-darwin-arm64
./autoscan-darwin-arm64
```

**Linux:**
```bash
chmod +x autoscan-linux-amd64
./autoscan-linux-amd64
```

On first run, autOScan auto-installs to `~/.local/bin/autoscan` and prompts for PATH updates if needed.

### Build From Source

```bash
git clone https://github.com/autoscan-lab/autOScan.git
cd autOScan
make install
```

Requires: Go 1.22+, `gcc`

---

## Quickstart

```bash
autoscan
```

1. Choose **Run Grader**.
2. Pick a policy.
3. Select the submissions folder.
4. Review results and run tests if needed.

---

## Configuration

On first run, defaults are created at:

```text
~/.config/autoscan/
├── policies/
├── libraries/
├── test_files/
├── expected_outputs/
├── banned.yaml
├── ai_dictionary.yaml
└── settings.yaml
```

### Settings

- `keep_binaries`: keep compiled binaries for the **Run** tab
- `max_workers`: cap parallel compile workers (`0` = all cores)
- `plagiarism_*`: similarity detection tuning
- `ai_*`: AI-pattern detection tuning

---

## Policy Basics

### Single-Process Example

```yaml
name: "Lab 03"
compile:
  gcc: "gcc"
  flags: ["-Wall", "-Wextra", "-lpthread"]
  source_file: "lab03.c"
run:
  test_cases:
    - name: "Basic test"
      args: ["2", "3"]
      expected_exit: 0
library_files:
  - hospital.h
  - hospital.o
test_files:
  - input.txt
```

### Multi-Process Example

```yaml
name: "Lab 07 - Message Queues"
compile:
  gcc: "gcc"
  flags: ["-Wall"]
run:
  multi_process:
    enabled: true
    executables:
      - name: "Producer"
        source_file: "producer.c"
        args: ["queue1", "5"]
      - name: "Consumer"
        source_file: "consumer.c"
        args: ["queue1"]
        start_delay_ms: 100
    test_scenarios:
      - name: "No args"
        process_args:
          Producer: []
          Consumer: []
        expected_exits:
          Producer: 1
          Consumer: 1
```

### Bundled Files

When added through the policy editor, files are copied into config directories so policies work from any working directory:

- `library_files` -> `~/.config/autoscan/libraries/`
- `test_files` -> `~/.config/autoscan/test_files/`
- expected outputs -> `~/.config/autoscan/expected_outputs/`

---

## Running Submissions

The **Run** tab supports:

- Manual execution with custom args/stdin
- Running policy test cases
- Multi-process execution with real-time output
- Expected output diff checks (`expected_output_file` / `expected_outputs`)

Requirement: enable `keep_binaries` before grading if you want to execute compiled programs in detail view.

---

## License

MIT
