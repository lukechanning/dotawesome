local conform = require('conform')
conform.setup({
  notify_on_error = true,
  notify_no_formatters = false,
  formatters_by_ft = {
    javascript = { 'prettier' }, typescript = { 'prettier' },
    javascriptreact = { 'prettier' }, typescriptreact = { 'prettier' },
    json = { 'prettier' }, jsonc = { 'prettier' },
    lua = { 'stylua' },
  },
  format_on_save = function(buf)
    local available = false
    for _, formatter in ipairs(conform.list_formatters(buf)) do
      available = available or formatter.available
    end
    if available then
      return { timeout_ms = 1000, lsp_format = 'never' }
    end
  end,
})
vim.keymap.set('n', '<leader>f', function()
  local formatters = conform.list_formatters(0)
  if not vim.iter(formatters):any(function(f) return f.available end) then
    vim.notify('No formatter available. Install prettier or stylua with :Mason.', vim.log.levels.WARN)
    return
  end
  conform.format({ async = true, lsp_format = 'never' })
end, { desc = 'Format buffer' })
local lint = require('lint')
lint.linters_by_ft = {
  javascript = { 'eslint' }, typescript = { 'eslint' },
  javascriptreact = { 'eslint' }, typescriptreact = { 'eslint' },
}
vim.api.nvim_create_autocmd({ 'BufWritePost', 'InsertLeave' }, {
  group = vim.api.nvim_create_augroup('DotawesomeLint', { clear = true }),
  callback = function()
    if vim.fn.executable('eslint') == 1 then
      lint.try_lint()
    end
  end,
})
