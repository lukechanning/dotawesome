" Always show line numbers
set number
set updatetime=300
set splitbelow
set splitright
let mapleader =" "
" Setup Plugins through vim-plug
call plug#begin('~/.vim/plugged')
Plug 'sonph/onehalf', {'rtp': 'vim/'}
Plug 'vim-airline/vim-airline'
Plug 'vim-airline/vim-airline-themes'
Plug 'christoomey/vim-tmux-navigator' 
Plug 'benmills/vimux'
Plug 'preservim/nerdtree'
Plug 'sheerun/vim-polyglot'
Plug 'tpope/vim-surround'
Plug 'junegunn/fzf', { 'do': './install --bin' }
Plug 'junegunn/fzf.vim'
Plug 'airblade/vim-gitgutter'
Plug 'tpope/vim-commentary'
Plug 'neoclide/coc.nvim', {'branch': 'release'}
call plug#end()
" End Plugin setup

" ==One Half== "
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

" === FZF Var === "
let $FZF_DEFAULT_COMMAND = 'rg --files --hidden --follow -g "!.git/*"'

" === NERDTree === "
" Show hidden files/directories
let g:NERDTreeShowHidden = 1
" Remove bookmarks and help text from NERDTree
let g:NERDTreeMinimalUI = 1
" Hide certain files and directories from NERDTree
let g:NERDTreeIgnore = ['^\.DS_Store$', '^tags$', '\.git$[[dir]]', '\.idea$[[dir]]', '\.sass-cache$']

" Space Handling
filetype plugin indent on
set expandtab
set tabstop=2
set softtabstop=2
set shiftwidth=2

" === Coc.nvim === "
" use <tab> for trigger completion and navigate to next complete item
function! s:check_back_space() abort
  let col = col('.') - 1
  return !col || getline('.')[col - 1]  =~ '\s'
endfunction
inoremap <silent><expr> <TAB>
      \ pumvisible() ? "\<C-n>" :
      \ <SID>check_back_space() ? "\<TAB>" :
      \ coc#refresh()
"Close preview window when completion is done.
autocmd! CompleteDone * if pumvisible() == 0 | pclose | endif
" Use <cr> to confirm completion, `<C-g>u` means break undo chain at current position.
" Coc only does snippet and additional edit on confirm.
inoremap <expr> <cr> pumvisible() ? "\<C-y>" : "\<C-g>u\<CR>"
" Use `:Format` to format current buffer
" command! -nargs=0 Format :call CocAction('format')
" Use Ctrl-S-F to run linter
nnoremap <leader><C-F> :CocCommand eslint.executeAutofix<CR>
" Use `[g` and `]g` to navigate diagnostics
nmap <silent> [g <Plug>(coc-diagnostic-prev)
nmap <silent> ]g <Plug>(coc-diagnostic-next)
" Remap keys for gotos
nmap <silent> gd <Plug>(coc-definition)
nmap <silent> gi <Plug>(coc-implementation)
nmap <silent> gr <Plug>(coc-references)
" Remap for do codeAction of current line
nmap <leader>ac  <Plug>(coc-codeaction)

" ==NERDTree== "
map <C-n> :NERDTreeToggle<CR>

" ==Remapped Keys== "
" fuzzy search
nnoremap <C-p> :Files<CR>
" Remap split navigation

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
