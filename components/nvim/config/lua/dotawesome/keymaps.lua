local map = vim.keymap.set
map('n', 'vv', '<C-w>v', { desc = 'Split vertically' })
map('n', '<C-w>+', '<cmd>resize +10<CR>')
map('n', '<C-w>-', '<cmd>resize -10<CR>')
map('n', '<C-w>>', '<cmd>vertical resize +10<CR>')
map('n', '<C-w><', '<cmd>vertical resize -10<CR>')
map('t', '<Esc>', '<C-\\><C-n>', { desc = 'Leave terminal mode' })
for key, direction in pairs({ h = 'h', j = 'j', k = 'k', l = 'l' }) do
  map('t', '<C-' .. key .. '>', '<C-\\><C-n><C-w>' .. direction)
  map('i', '<C-' .. key .. '>', '<Esc><C-w>' .. direction)
end
