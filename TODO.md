# Future Improvements

## High Priority

- [ ] **TUI view refactor**
  - Split `internal/tui/views.go` and `handlers.go` into smaller files grouped by view (home, policy manage/editor, submissions/results, similarity, details/run, export)
  - Keep shared styling/helpers in one place and remove redundant comments (should explain why and not what)
  - Plan: create view/handler files per screen, keep shared helpers in `tui_common.go`

- [ ] **AI-generated code detection**
  - Build an AI-pattern dictionary and score submissions using the similarity pipeline

## Low Priority

- [ ] **Valgrind Integration** - Memory leak detection (pass/fail)
