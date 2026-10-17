vim.g.mapleader = ' '
vim.g.maplocalleader = ' '
local opt = vim.opt
opt.number = true
opt.updatetime = 300
opt.splitbelow = true
opt.splitright = true
opt.expandtab = true
opt.tabstop = 2
opt.softtabstop = 2
opt.shiftwidth = 2
opt.termguicolors = true
opt.undofile = true
opt.undodir = vim.fn.stdpath('state') .. '/undo'
vim.fn.mkdir(opt.undodir:get()[1], 'p')
opt.ignorecase = true
opt.smartcase = true
opt.signcolumn = 'yes'
opt.completeopt = { 'menu', 'menuone', 'noselect' }
opt.statusline = ' %f %h%m%r%=%{&filetype} %l:%c '
vim.g.edge_better_performance = 1
