# Future Improvements

## High Priority

- [ ] **TUI view refactor** (in progress)
  - [x] Create `views/settings/` subpackage
  - [x] Create `views/home/` subpackage
  - [x] Create `views/policy/` subpackage (select.go, manage.go, editor.go)
  - [x] Create `views/banned/` subpackage
  - [x] Create `views/directory/` subpackage
  - [ ] Create `views/submissions/` subpackage (largest)
  - [ ] Create `views/details/` subpackage
  - [ ] Create `views/export/` subpackage
  - [ ] Extract repeated patterns (scrollable lists, info blocks, tab bars)

- [ ] **AI-generated code detection**
  - Build an AI-pattern dictionary and score submissions using the similarity pipeline

## Low Priority

- [ ] **Valgrind Integration** - Memory leak detection (pass/fail)
