# Restored executables and mise; keep the installer shell configuration.
fish_add_path --path $HOME/.local/bin
if status is-interactive; and command -q mise
    mise activate fish | source
end
