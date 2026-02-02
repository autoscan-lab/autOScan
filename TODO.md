# Future Improvements

## High Priority

- [x] **TUI view refactor** ✓ Complete
  - [x] Create `views/settings/` subpackage
  - [x] Create `views/home/` subpackage
  - [x] Create `views/policy/` subpackage (select.go, manage.go, editor.go)
  - [x] Create `views/banned/` subpackage
  - [x] Create `views/directory/` subpackage
  - [x] Create `views/export/` subpackage
  - [x] Create `views/submissions/` subpackage (submissions.go, helpers.go)
  - [x] Create `views/details/` subpackage (details.go, helpers.go, compile.go, banned.go, files.go, run.go)
  - [x] Extract common utilities to `components/common.go` (path truncation, rendering helpers)

- [ ] **AI-generated code detection**
  - Build an AI-pattern dictionary and score submissions using the similarity pipeline

## Low Priority

- [ ] **Valgrind Integration** - Memory leak detection (pass/fail)
