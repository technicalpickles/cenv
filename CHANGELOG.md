# Changelog

## [0.2.0](https://github.com/technicalpickles/cenv/compare/v0.1.0...v0.2.0) (2026-07-02)


### Features

* **auth:** add auth detection package ([75ae193](https://github.com/technicalpickles/cenv/commit/75ae193ad3335f290dc2a13ce16b7123ad3d70fd))
* **bootstrap:** add bootstrap package for env initialization ([f25ff05](https://github.com/technicalpickles/cenv/commit/f25ff053ab619f1d259f86cfbce119402be8f92e))
* **cli:** add auth create and auth list subcommands ([53b4cad](https://github.com/technicalpickles/cenv/commit/53b4cade99858043ca433dd5a2282d68f8f1f5d8))
* **cli:** add settings subcommands (show, get, merge) ([bb521d0](https://github.com/technicalpickles/cenv/commit/bb521d070f2bef57264fe23447d06eb4af145cc5))
* **cmd:** add 'cenv login' for interactive OAuth auth ([2361490](https://github.com/technicalpickles/cenv/commit/23614907b23d6ac9b183dab9e90b7ac009214c87))
* **cmd:** add create, list, remove, path commands ([4dfb5d6](https://github.com/technicalpickles/cenv/commit/4dfb5d69f385255ae3be1a17b0b14998ebb9c21b))
* **cmd:** add isTerminal helper ([4fd21b7](https://github.com/technicalpickles/cenv/commit/4fd21b744a611774b26571970d6f4b36c4e58a26))
* **cmd:** add run command to launch Claude in an isolated environment ([cde6074](https://github.com/technicalpickles/cenv/commit/cde60741d73e8703ce19c97a58389839cb1f1a49))
* **cmd:** point OAuth users at cenv login on create ([d5cc99a](https://github.com/technicalpickles/cenv/commit/d5cc99a89827cb9ff7771d05f4a4692650d0870a))
* **cmd:** pre-flight auth detection in cenv run ([1a363c0](https://github.com/technicalpickles/cenv/commit/1a363c07ed5cd336dc2f4d63b66eaa81a8fb76ec))
* **cmd:** refuse 'auth create' for OAuth users ([3fa07fd](https://github.com/technicalpickles/cenv/commit/3fa07fd6fa2a4328e56212a2449ba1620ccb3a66))
* **env:** add internal/env package with core CRUD operations ([4f9e208](https://github.com/technicalpickles/cenv/commit/4f9e208f7a47640325c770fceb38f8c90a862308))
* polish pass for programmatic callers (gt-z80p) ([4d2c417](https://github.com/technicalpickles/cenv/commit/4d2c4173046d989e7d3157381e48904285ba5ea1))
* project scaffolding with cobra root command ([976972a](https://github.com/technicalpickles/cenv/commit/976972a19dacfba40be76c766b302ad50c955387))
* **settings:** add DeepMerge for recursive map merging ([79a4896](https://github.com/technicalpickles/cenv/commit/79a4896bd4eeac7b9002df27bb192ae4184e1818))
* **settings:** add GetByDotPath for dot-separated key traversal ([d6a6aaa](https://github.com/technicalpickles/cenv/commit/d6a6aaa2b8f4393ebf6000d674cb93f48c3b72d0))
* **settings:** add JSON detection and settings file operations ([eb4c10c](https://github.com/technicalpickles/cenv/commit/eb4c10ce0abf50e7b7a7eee0630483aa7f3e5ec1))


### Bug Fixes

* **auth:** handle object-shaped oauthAccount (gt-5zn7) ([c3abfbf](https://github.com/technicalpickles/cenv/commit/c3abfbfead4da4a3c780a142f4aee698eb26b77e))
* **cmd:** hasOAuth accepts object-shaped oauthAccount ([08fb9b4](https://github.com/technicalpickles/cenv/commit/08fb9b4734473f5b410e31b0297c926103aae603))
