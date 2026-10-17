local M = {}
function M.check()
  vim.health.start('Dotawesome editor tools')
  for _, command in ipairs({ 'git', 'rg' }) do
    if vim.fn.executable(command) == 1 then
      vim.health.ok(command .. ' available')
    else
      vim.health.error(command .. ' missing')
    end
  end
  for command, purpose in pairs({
    ['typescript-language-server'] = 'TypeScript/JavaScript LSP (:Mason)',
    ['vscode-json-language-server'] = 'JSON LSP (:Mason)',
    prettier = 'JS/TS/JSON formatting (:Mason)',
    stylua = 'Lua formatting (:Mason)',
    eslint = 'JS/TS linting (project or :Mason)',
    ['tree-sitter'] = 'parser installation (Arch: tree-sitter-cli)',
    cc = 'parser compilation (Arch: gcc)',
    node = 'JavaScript tools (Arch: nodejs npm)',
  }) do
    if vim.fn.executable(command) == 1 then
      vim.health.ok(command .. ' available')
    else
      vim.health.warn(command .. ' missing; optional: ' .. purpose)
    end
  end
end
return M
