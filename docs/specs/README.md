
## Full-Screen TUI with Scrolling (2025-11-24)

Enhanced management mode to take over the full terminal screen with viewport-based scrolling.

**Status**: Ready for Implementation  
**Complexity**: 3/10 (Simple)  
**Estimated Effort**: 2-3 hours

### Documents

1. **[Summary](./tui-fullscreen-scrolling-summary.md)** - Start here! Overview and quick links
2. **[Main Spec](./tui-fullscreen-scrolling-spec.md)** - Complete feature specification  
3. **[Code References](./tui-fullscreen-scrolling-code-refs.md)** - Exact before/after code changes
4. **[Test Plan](./tui-fullscreen-scrolling-tests.md)** - Unit, integration, and manual tests
5. **[Architecture](./tui-fullscreen-scrolling-architecture.md)** - Design decisions and patterns
6. **[Validation](./tui-fullscreen-scrolling-VALIDATION.md)** - Pre-implementation validation report

### Key Changes

- Add viewport tracking to `CompletionModel`
- Implement viewport-aware rendering in `View()`
- Enable alt screen buffer with `tea.WithAltScreen()`
- Add scroll indicators (↑/↓)

### Implementation Phases

1. **Phase 1**: Add viewport fields (30 min)
2. **Phase 2**: Update navigation logic (45 min)
3. **Phase 3**: Refactor rendering (60 min)
4. **Phase 4**: Enable alt screen (5 min)

---

