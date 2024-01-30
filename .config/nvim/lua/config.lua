-------------------
-- Telescope ------
-------------------
require("telescope").setup {
  defaults = {
    file_ignore_patterns = { ".git/", "node_modules/" }
  },
  pickers = {
    find_files = {
      hidden = true
    }
  }
}

-------------------
-- TreeSitter -----
-------------------
local configs = require("nvim-treesitter.configs")
configs.setup {
  ensure_installed = {
    "jsonc",
    "json",
    "lua",
    "javascript",
    "typescript",
    "yaml",
    "html",
    "css",
    "bash",
    "tsx",
    "toml",
    "graphql",
    "rust"
  },
}

-------------------
-- NVIM Tree ------
-------------------
-- disable netrw at the very start of your init.lua
vim.g.loaded_netrw = 1
vim.g.loaded_netrwPlugin = 1

-- enable 24-bit colour
vim.opt.termguicolors = true

-- empty setup using defaults
require("nvim-tree").setup()

-----------------
-- LSP Setup ----
-----------------
-- mason config for lsp with deps
require("mason").setup()
require("mason-lspconfig").setup({
    ensure_installed = {
        "tsserver",
        "jsonls"
    }
})

-- Configure Language Servers
require("lspconfig").tsserver.setup {}
require("lspconfig").jsonls.setup {}

-- Configure Linting
require('lint').linters_by_ft = {
    javascript = {"eslint"},
    typescript = {"eslint"},
    javascriptreact = {"eslint"},
    typescriptreact = {"eslint"},
}

vim.api.nvim_create_autocmd({ "InsertLeave", "BufWritePost" }, {
  callback = function()
    require("lint").try_lint()
  end,
})

-- Configure Formatting
require('formatter').setup({
    logging = false,
    filetype = {
        typescript = {
            require("formatter.filetypes.typescript").prettier
        },
        javascript = {
            require("formatter.filetypes.javascript").prettier
        },
        typescriptreact = {
            require("formatter.filetypes.typescriptreact").prettier
        },
        javascriptreact = {
            require("formatter.filetypes.javascriptreact").prettier
        },
        lua = {
            require("formatter.filetypes.lua").stylua
        },
        json = {
            require("formatter.filetypes.json").prettier
        },
    } 
})

-- Additional LSP keybindings
local opts = { noremap = true, silent = true }
vim.api.nvim_set_keymap("n", "gd", "<cmd>lua vim.lsp.buf.definition()<CR>", opts)
vim.api.nvim_set_keymap("n", "K", "<cmd>lua vim.lsp.buf.hover()<CR>", opts)
