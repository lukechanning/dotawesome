local group = vim.api.nvim_create_augroup('DotawesomeLsp', { clear = true })
vim.api.nvim_create_autocmd('LspAttach', {
  group = group,
  callback = function(event)
    local client = vim.lsp.get_client_by_id(event.data.client_id)
    if client and client:supports_method('textDocument/completion') then
      vim.lsp.completion.enable(true, client.id, event.buf, { autotrigger = true })
    end
    local opts = { buffer = event.buf }
    vim.keymap.set('n', 'gd', vim.lsp.buf.definition, opts)
    vim.keymap.set('n', 'gr', vim.lsp.buf.references, opts)
    vim.keymap.set('n', 'K', vim.lsp.buf.hover, opts)
    vim.keymap.set('n', '<leader>rn', vim.lsp.buf.rename, opts)
    vim.keymap.set('n', '<leader>ca', vim.lsp.buf.code_action, opts)
  end,
})
require('mason').setup()
-- Installation is explicit through :Mason, never during configuration validation.
for name, command in pairs({ ts_ls = 'typescript-language-server', jsonls = 'vscode-json-language-server' }) do
  if vim.fn.executable(command) == 1 then
    vim.lsp.enable(name)
  end
end
