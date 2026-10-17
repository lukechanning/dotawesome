require('dotawesome.options')
require('dotawesome.keymaps')

-- Validation never bootstraps plugins or touches the real editor state.
if vim.env.DOTAWESOME_NVIM_VALIDATE == '1' then
  return
end
require('dotawesome.plugins')
