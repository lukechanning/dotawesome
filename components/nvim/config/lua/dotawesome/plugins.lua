local pins = require('dotawesome.pins')
local lazypath = vim.fn.stdpath('data') .. '/lazy/lazy.nvim'
if not vim.uv.fs_stat(lazypath) then
  local output = vim.fn.system({ 'git', 'clone', '--filter=blob:none',
    'https://github.com/folke/lazy.nvim.git', lazypath })
  if vim.v.shell_error ~= 0 then
    vim.notify('Plugin bootstrap failed; basic editing remains available.\n' .. output, vim.log.levels.ERROR)
    return
  end
end
-- Fetch only when a deliberately changed bootstrap pin is not available locally.
local revision = pins['folke/lazy.nvim']
vim.fn.system({ 'git', '-C', lazypath, 'cat-file', '-e', revision })
if vim.v.shell_error ~= 0 then
  local output = vim.fn.system({ 'git', '-C', lazypath, 'fetch', '--depth=1', 'origin', revision })
  if vim.v.shell_error ~= 0 then
    vim.notify('Cannot fetch pinned lazy.nvim revision.\n' .. output, vim.log.levels.ERROR)
    return
  end
end
local current = vim.fn.system({ 'git', '-C', lazypath, 'rev-parse', 'HEAD' }):gsub('%s+$', '')
if current ~= revision then
  local output = vim.fn.system({ 'git', '-C', lazypath, 'checkout', '--detach', revision })
  if vim.v.shell_error ~= 0 then
    vim.notify('Cannot select pinned lazy.nvim revision.\n' .. output, vim.log.levels.ERROR)
    return
  end
end
vim.opt.rtp:prepend(lazypath)
local function plugin(repo, options)
  local spec = options or {}
  spec[1] = repo
  spec.commit = assert(pins[repo], 'Missing plugin revision: ' .. repo)
  return spec
end
require('lazy').setup({
  plugin('sainnhe/edge', { priority = 1000, config = function() vim.cmd.colorscheme('edge') end }),
  plugin('nvim-lua/plenary.nvim'),
  plugin('nvim-tree/nvim-web-devicons'),
  plugin('nvim-telescope/telescope.nvim', {
    dependencies = { 'plenary.nvim' },
    config = function()
      require('telescope').setup({
        defaults = { file_ignore_patterns = { '%.git/', 'node_modules/' } },
        pickers = { find_files = { hidden = true } },
      })
      local builtin = require('telescope.builtin')
      for key, action in pairs({ ['<leader>ff'] = 'find_files', ['<leader>fg'] = 'live_grep',
        ['<leader>fb'] = 'buffers', ['<leader>fh'] = 'help_tags', ['<C-p>'] = 'find_files' }) do
        vim.keymap.set('n', key, builtin[action], { desc = action:gsub('_', ' ') })
      end
    end,
  }),
  plugin('nvim-tree/nvim-tree.lua', {
    dependencies = { 'nvim-web-devicons' },
    config = function()
      vim.g.loaded_netrw = 1
      vim.g.loaded_netrwPlugin = 1
      require('nvim-tree').setup()
      vim.keymap.set('n', '<C-n>', '<cmd>NvimTreeToggle<CR>', { desc = 'Toggle file tree' })
    end,
  }),
  plugin('christoomey/vim-tmux-navigator'),
  plugin('preservim/vimux', { config = function()
    vim.keymap.set('n', '<leader>T', '<cmd>VimuxPromptCommand<CR>', { desc = 'Run command in tmux' })
  end }),
  plugin('tpope/vim-surround'),
  plugin('tpope/vim-commentary'),
  plugin('windwp/nvim-autopairs', { opts = {} }),
  plugin('lewis6991/gitsigns.nvim', { opts = {} }),
  plugin('mason-org/mason.nvim'),
  plugin('neovim/nvim-lspconfig', {
    dependencies = { 'mason.nvim' },
    config = function() require('dotawesome.lsp') end,
  }),
  plugin('mfussenegger/nvim-lint'),
  plugin('stevearc/conform.nvim', {
    dependencies = { 'nvim-lint' },
    config = function() require('dotawesome.formatting') end,
  }),
  plugin('nvim-treesitter/nvim-treesitter', {
    lazy = false,
    config = function() require('dotawesome.treesitter') end,
  }),
  plugin('github/copilot.vim', { enabled = vim.env.DOTAWESOME_COPILOT == '1' }),
}, {
  defaults = { lazy = false },
  lockfile = vim.fn.stdpath('state') .. '/lazy-lock.json',
  local_spec = false,
  checker = { enabled = false },
  change_detection = { notify = false },
  rocks = { enabled = false },
})
