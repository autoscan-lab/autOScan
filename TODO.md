# Future Improvements

## High Priority

### Plagiarism Detection (Tree-sitter window-hash algorithm)

- [ ] **TUI pair detail view**
  - Side-by-side views: left file A, right file B
  - Highlight matched spans; keyboard navigation between matches
  - Summary header: window Jaccard, per-function similarity, match counts

## Low Priority

- [ ] **TUI view refactor**
  - Split `internal/tui/views.go` and `handlers.go` into smaller files grouped by view (home, policy manage/editor, submissions/results, similarity, details/run, export)
  - Keep shared styling/helpers in one place and remove redundant comments (should explain why and not what)

- [ ] **Valgrind Integration** - Memory leak detection (pass/fail)
