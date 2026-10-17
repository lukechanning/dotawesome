require('nvim-treesitter').setup()
vim.api.nvim_create_autocmd('FileType', {
  group = vim.api.nvim_create_augroup('DotawesomeTreesitter', { clear = true }),
  callback = function(event)
    local lang = vim.treesitter.language.get_lang(vim.bo[event.buf].filetype)
    if not lang then return end
    local ok, loaded = pcall(vim.treesitter.language.add, lang)
    if ok and loaded then
      vim.treesitter.start(event.buf, lang)
      if vim.treesitter.query.get(lang, 'indents') then
        vim.bo[event.buf].indentexpr = "v:lua.require'nvim-treesitter'.indentexpr()"
      end
    end
  end,
})
