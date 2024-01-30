-- Mason Setup
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