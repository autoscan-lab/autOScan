# Release Notes

## Major Features

### 🚀 Library Compilation
- **Automated library compilation**: Add instructor-provided library files (`.c`, `.h`, `.o`) to policies
- **Bundled libraries**: Library files are automatically copied to `~/.config/autoscan/libraries/` for portability
- **Smart compilation**: GCC automatically handles `.h` files via `#include` and links `.o` files correctly
- **Library management**: Add libraries from working directory or select from existing bundled libraries
- **Proper flag ordering**: Compiler and linker flags are automatically ordered correctly for GCC

### ▶️ Run & Execute Feature
- **Interactive execution**: Run compiled binaries directly from the TUI with custom arguments
- **Stdin support**: Provide input via stdin with support for newlines (`\n`)
- **Test cases**: Define preset test cases in policies with arguments, stdin, and expected exit codes
- **Output display**: View stdout, stderr, exit codes, and execution time
- **Timeout handling**: Automatic timeout for long-running processes
- **Scrollable output**: Full output display with scrolling (no truncation)

### 🔄 Multi-Process Execution
- **Parallel execution**: Run multiple binaries simultaneously (e.g., producer/consumer patterns)
- **Process configuration**: Define separate processes with individual source files, arguments, and delays
- **Grid layout**: Responsive 2-column grid layout for process outputs
- **Process management**: Start delays, timeout handling, and individual process status
- **Signal handling**: SIGKILL button for terminating deadlocked processes
- **Visual feedback**: Color-coded borders (green for pass, red for fail) and status indicators

### 📋 Multi-Process Test Scenarios
- **Multiple test configurations**: Define different test scenarios for multi-process labs
- **Per-process overrides**: Each scenario can override arguments, stdin, and expected exit codes per process
- **Quick access**: Run scenarios with number keys `[1-9]` in the Run tab
- **Scenario naming**: Descriptive names for different test cases

### 📁 Bundled Test Files
- **Input file management**: Add test input files (`.txt`, `.bin`, etc.) to policies
- **Automatic path resolution**: Test file names in arguments are automatically resolved to full paths
- **Bundled storage**: Test files are copied to `~/.config/autoscan/test_files/` for portability
- **File selection**: Add new files or select from existing bundled files

## UI Improvements

### 📐 Responsive Layout
- **Full-width utilization**: Application now uses the entire terminal width (like Claude Code)
- **Dynamic sizing**: All components adapt to terminal size with minimum width constraints
- **Better readability**: Wider columns and boxes prevent text wrapping
- **Responsive tables**: Submission table columns adjust based on available width

### ✏️ Enhanced Policy Editor
- **Two-column layout**: Better horizontal space utilization with organized grid layout
- **Inline editing**: Edit test cases and processes directly from the list view
- **Visual indicators**: Clear hints showing when fields are relevant (e.g., "If Multi-Process OFF")
- **Improved navigation**: Arrow keys properly navigate within lists before moving between fields
- **Scroll indicators**: Shows "1-5 of 9" when lists exceed visible area
- **Context-aware hints**: Global help bar updates based on focused section

### 🎨 Visual Polish
- **Consistent alignment**: Fixed indentation issues across all list sections
- **Better focus indicators**: Clear visual feedback for editable sections
- **Prominent buttons**: Save/Cancel buttons with color-coded backgrounds
- **Status indicators**: Clear pass/fail indicators in multi-process results
- **No truncation**: Compile output and execution results show full content with wrapping

## Technical Improvements

### 🔧 Compilation Engine
- **Better error handling**: Improved validation and warnings for library files
- **Object file validation**: Checks for empty or corrupted `.o` files
- **Flag ordering**: Proper separation of compiler flags and linker flags
- **Include path handling**: Automatic `-I` flag addition for header file discovery
- **Multi-process compilation**: Each source file compiles to its own binary

### 🗂️ State Management
- **Proper state clearing**: Results are cleared when switching submissions or policies
- **Executor recreation**: Ensures executor uses current policy configuration
- **No stale data**: Previous execution results don't persist across policy/submission changes

## Configuration

### Policy Structure
Policies now support:
- `library_files`: List of instructor-provided files (`.c`, `.h`, `.o`)
- `test_files`: List of input files for testing
- `run.multi_process`: Multi-process configuration with executables and test scenarios
- `run.test_cases`: Preset test cases with arguments, stdin, and expected exits

### Directory Structure
- `~/.config/autoscan/libraries/`: Bundled library files
- `~/.config/autoscan/test_files/`: Bundled test input files
- `~/.config/autoscan/policies/`: Policy YAML files

## Bug Fixes

- Fixed column alignment issues in tables
- Fixed list item shifting when items are selected
- Fixed ESC key exiting entire editor instead of just sub-modes
- Fixed multi-process compilation failing for all submissions
- Fixed overlapping process output boxes in multi-process grid
- Fixed stale execution results showing from previous policies
- Fixed compilation output truncation
- Fixed `.o` file handling in compilation

## Migration Notes

- Existing policies will continue to work
- New fields (`library_files`, `test_files`, `multi_process`) are optional
- Library files can be added via the policy editor UI
- Test files can be added via the policy editor UI

---

**Note**: This release includes significant UI and functionality improvements. The application is now more powerful and user-friendly while maintaining backward compatibility with existing policies.
