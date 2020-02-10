" Always show line numbers
set number
" Setup Plugins through vim-plug
call plug#begin('~/.vim/plugged')
Plug 'sonph/onehalf', {'rtp': 'vim/'}
Plug 'vim-airline/vim-airline'
Plug 'vim-airline/vim-airline-themes'
Plug 'preservim/nerdtree'
Plug 'sheerun/vim-polyglot'
Plug 'junegunn/fzf', { 'do': './install --bin' }
Plug 'junegunn/fzf.vim'
Plug 'airblade/vim-gitgutter'
Plug 'tpope/vim-commentary'
Plug 'neoclide/coc.nvim', {'branch': 'release'}
call plug#end()
" End Plugin setup

syntax on
set t_Co=256
set cursorline
colorscheme onehalfdark
let g:airline_theme='onehalfdark'

if exists('+termguicolors')
  let &t_8f = "\<Esc>[38;2;%lu;%lu;%lum"
  let &t_8b = "\<Esc>[48;2;%lu;%lu;%lum"
  set termguicolors
endif

" Space Handling
filetype plugin indent on
set expandtab
set tabstop=2
set softtabstop=2
set shiftwidth=2

" Remapped Keys
map <C-n> :NERDTreeToggle<CR>
" fuzzy search
nnoremap <C-p> :Files<CR>
" Remap split navigation
nnoremap <C-J> <C-W><C-J>
nnoremap <C-K> <C-W><C-K>
nnoremap <C-L> <C-W><C-L>
nnoremap <C-H> <C-W><C-H>

" Remap sizing
noremap <C-w>+ :resize +10<CR>
noremap <C-w>- :resize -10<CR>
noremap <C-w>> :vertical:resize -10<CR>
noremap <C-w>< :vertical:resize +10<CR>

