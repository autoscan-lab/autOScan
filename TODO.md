# Future Improvements

## High Priority

- [x] **Student Search/Filter**
  - Press `/` or `↑` to filter submissions by name
  - Real-time filtering as you type

- [ ] **Code Similarity Detection**
  - AST-based comparison using tree-sitter
  - Compute automatically after scan completes
  - New "Similarity" tab in Run Grader
  - Table sorted by similarity percentage (highest first)
  - Side-by-side code diff viewer on selection
  - Flag pairs for review (flags both submissions)
  - Flags persist to Results tab and exports (JSON/CSV)
  - Configurable threshold (default 70%)

- [ ] **Expected Output Comparison**
  - Add `expected_output_file` to policy test cases
  - For multi-process: `expected_outputs` map per process in test scenarios
  - Git-style inline diff view in Run tab
  - Exact match: show "PASS - Matches expected output"
  - Mismatch: show "CHECK" with diff count and inline diff
  - Reference files stored in test_files/
  - Per-process diff display for multi-process execution

## Low Priority

- [ ] **Valgrind Integration** - Memory leak detection (pass/fail)
