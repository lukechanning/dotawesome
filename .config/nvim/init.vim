" Always show line numbers
set number
set updatetime=300
set splitbelow
set splitright
let mapleader =" "
" Setup Plugins through vim-plug
call plug#begin('~/.config/nvim/plugged')
Plug 'dracula/vim', { 'as': 'dracula' }
Plug 'vim-airline/vim-airline'
Plug 'christoomey/vim-tmux-navigator'
Plug 'benmills/vimux'
Plug 'preservim/nerdtree'
Plug 'sheerun/vim-polyglot'
Plug 'tpope/vim-surround'
Plug 'jiangmiao/auto-pairs'
Plug 'junegunn/fzf', { 'do': './install --all' }
Plug 'junegunn/fzf.vim'
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
colorscheme dracula
set termguicolors
highlight Visual cterm=reverse ctermbg=NONE

" === FZF Var === "
let $FZF_DEFAULT_COMMAND = 'rg --files --hidden --follow -g "!.git/*"'

" === Mason, LSP, Linting & Formatting === "
lua require("config")
" Format on save
augroup FormatAutogroup
  autocmd!
  autocmd BufWritePost * FormatWrite
augroup END

" === NERDTree === "
" Show hidden files/directories
let g:NERDTreeShowHidden = 1
" Remove bookmarks and help text from NERDTree
let g:NERDTreeMinimalUI = 1
" Hide certain files and directories from NERDTree
let g:NERDTreeIgnore = ['^\.DS_Store$', '^tags$', '\.git$[[dir]]', '\.idea$[[dir]]', '\.sass-cache$']
" OPen NERDTree with Ctrl-n
map <C-n> :NERDTreeToggle<CR>

" ==Remapped Keys== "
" fuzzy search
nnoremap <C-p> :Files<CR>

" Remap sizing
nnoremap <silent> vv <C-w>v
noremap <C-w>+ :resize +10<CR>
noremap <C-w>- :resize -10<CR>
noremap <C-w>> :vertical:resize -10<CR>
noremap <C-w>< :vertical:resize +10<CR>

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
