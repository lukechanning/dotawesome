" Always show line numbers
set number
set updatetime=300
set splitbelow
set splitright
let mapleader =" "
" Setup Plugins through vim-plug
call plug#begin('~/.config/nvim/plugged')
" Plug 'dracula/vim', { 'as': 'dracula' }
Plug 'projekt0n/github-nvim-theme'
Plug 'vim-airline/vim-airline'
Plug 'christoomey/vim-tmux-navigator'
Plug 'benmills/vimux'
Plug 'nvim-tree/nvim-web-devicons'
Plug 'nvim-tree/nvim-tree.lua'
Plug 'nvim-treesitter/nvim-treesitter', {'do': ':TSUpdate'}
Plug 'tpope/vim-surround'
Plug 'jiangmiao/auto-pairs'
Plug 'nvim-lua/plenary.nvim'
Plug 'nvim-telescope/telescope.nvim', { 'tag': '0.1.5' }
Plug 'airblade/vim-gitgutter'
Plug 'tpope/vim-commentary'
Plug 'williamboman/mason.nvim'
Plug 'williamboman/mason-lspconfig.nvim'
Plug 'neovim/nvim-lspconfig'
Plug 'mfussenegger/nvim-lint'
Plug 'mhartington/formatter.nvim'
Plug 'github/copilot.vim'
call plug#end()
" End Plugin setup

" == Raw VIM config =="
set undofile " Maintain undo history between sessions
set undodir=~/.config/nvim/undodir

" Space Handling
filetype plugin indent on
set expandtab
set tabstop=2
set softtabstop=2
set shiftwidth=2

" == Color Scheme Stuff =="
" colorscheme dracula
colorscheme github_light 
set termguicolors
highlight Visual cterm=reverse ctermbg=NONE

" == Lua Config == "
lua require("config")

" === Mason, LSP, Linting & Formatting === "
" Format on save
augroup FormatAutogroup
  autocmd!
  autocmd BufWritePost * FormatWrite
augroup END

" ==Remapped Keys== "
" Remap sizing
nnoremap <silent> vv <C-w>v
noremap <C-w>+ :resize +10<CR>
noremap <C-w>- :resize -10<CR>
noremap <C-w>> :vertical:resize -10<CR>
noremap <C-w>< :vertical:resize +10<CR>

" Open NVIM Tree with Ctrl-n
map <C-n> :NvimTreeToggle<CR>

" Telescope
nnoremap <leader>ff <cmd>Telescope find_files<cr>
nnoremap <leader>fg <cmd>Telescope live_grep<cr>
nnoremap <leader>fb <cmd>Telescope buffers<cr>
nnoremap <leader>fh <cmd>Telescope help_tags<cr>
nnoremap <C-p> <cmd>Telescope find_files<cr>

" == Tmux == "
map <Leader><S-T> :VimuxPromptCommand<CR>

if has('nvim')
  tnoremap <Esc> <C-\><C-n>
  "tnoremap <M-[> <Esc>
  " Terminal mode:
  tnoremap <C-h> <c-\><c-n><c-w>h
  tnoremap <C-j> <c-\><c-n><c-w>j
  tnoremap <C-k> <c-\><c-n><c-w>k
  tnoremap <C-l> <c-\><c-n><c-w>l
  " Insert mode:
  inoremap <C-h> <Esc><c-w>h
  inoremap <C-j> <Esc><c-w>j
  inoremap <C-k> <Esc><c-w>k
  inoremap <C-l> <Esc><c-w>l
  " Visual mode:
  vnoremap <C-h> <Esc><c-w>h
  vnoremap <C-j> <Esc><c-w>j
  vnoremap <C-k> <Esc><c-w>k
  vnoremap <C-l> <Esc><c-w>l
  " Normal mode:
  nnoremap <C-h> <c-w>h
  nnoremap <C-j> <c-w>j
  nnoremap <C-k> <c-w>k
  nnoremap <C-l> <c-w>l
endif
