# Neovim & tmux

```sh
bin/dotawesome plan nvim
bin/dotawesome apply nvim
bin/dotawesome plan tmux
bin/dotawesome apply tmux
```

Each component backs up existing managed files and restores them on `remove`.
Unrelated files, plugins, caches, undo history, and application data are retained.

## Neovim

Neovim 0.12+, Git, and ripgrep. Lua configuration with Edge, Telescope, nvim-tree,
native LSP completion, formatting/linting, and tmux navigation. Plugins download
on first launch through lazy.nvim. Failed bootstrap leaves basic editing available;
retry after fixing the reported network/Git issue.

If an existing `init.vim` is present, move it aside first: Neovim cannot use it
alongside `init.lua`. The CLI reports the conflict and does not delete it.

Optional tools are installed explicitly:

```vim
:MasonInstall typescript-language-server json-lsp prettier stylua
:TSInstall jsonc json lua javascript typescript yaml html css bash tsx toml graphql rust
:checkhealth dotawesome
```

Restart Neovim after installing language servers. For linting, put `eslint` on
your shell PATH (for example, `npm install -g eslint`) and use your project’s
ESLint configuration. JavaScript tools need Node.js/npm.
Parser installation needs `tree-sitter-cli` 0.26.1+, a C compiler, curl, and tar.
Missing optional tools leave basic editing usable; health checks explain gaps.
Formatting runs on save only when its formatter is available.

| Keys | Action |
| --- | --- |
| Space+ff / fg / fb / fh | Find files / grep / buffers / help |
| Ctrl+P / Ctrl+N | Find files / toggle tree |
| Ctrl+H/J/K/L | Navigate editor/tmux panes |
| `vv` | Vertical split |
| Space+T / Space+f | Vimux command / format buffer |
| `gd` / `gr` / `K` | Definition / references / hover (attached LSP) |
| Space+rn / Space+ca | Rename / code action (attached LSP) |
| `gc` / `ys` | Comment / surround |

Copilot is off by default. Start with `DOTAWESOME_COPILOT=1 nvim`, then use
`:Copilot setup` to opt in and authenticate.

Plugin revisions are pinned in `lua/dotawesome/pins.lua`. Updates are deliberate:
change pins in the repo, validate in a disposable editor, apply, then `:Lazy sync`.
The mutable Lazy lockfile lives in Neovim's state directory, outside managed files.
No automatic update checks or parser/tool downloads run at startup.

## tmux

Uses the current plugin-free Breeze configuration, mouse support, Ctrl+A prefix,
Vim-aware pane navigation, and existing split bindings. No TPM installation.

New sessions load the deployed configuration. To reload an existing session:

```sh
tmux source-file ~/.tmux.conf
```

Apply/remove do not reload or terminate running sessions. Validation uses its
own temporary server/socket.
