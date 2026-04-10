# nsh (Go CLI)

[English](README.md) | [繁體中文](README.zh-TW.md)

`nsh` 是一個 macOS 專用的 SSH host 管理 CLI（Go + Cobra + Bubble Tea）。

它會把受管 host 存在 `~/.ssh/nsh/config`，並自動在主 `~/.ssh/config` 注入 `Include ~/.ssh/nsh/config`。你可以用群組、描述、驗證方式與排序標籤管理 host，同時保留原始 SSH 設定格式。

## 功能重點

- 互動式 TUI：瀏覽、搜尋、連線、編輯、刪除、釘選
- Host 全生命週期：`new` / `copy` / `edit` / `del` / `auth`
- 匯入匯出：支援純設定與加密完整備份（含密碼、金鑰）
- 安全機制：Keychain、Touch ID、atomic write、備份輪替
- 相容主設定：保留 `Include`、`Match`、註解等既有內容

## 系統需求

- macOS（Keychain、Touch ID、`ssh-add --apple-use-keychain`）

## 安裝

```bash
brew tap kitdot/tap
brew install kitdot/tap/nsh
```

## 移除

```bash
brew uninstall nsh
brew untap kitdot/tap
```

## 從原始碼編譯

需要 Go 1.25+。

```bash
git clone https://github.com/kitdot/nsh.git
cd nsh
CGO_ENABLED=1 go build -ldflags "-X github.com/kitdot/nsh/cmd.nshVersion=0.1.0" -o nsh .
sudo install -m 0755 nsh /usr/local/bin/nsh
```

不注入版本時會顯示 `dev`。若要用 git tag 自動帶版本：

```bash
CGO_ENABLED=1 go build -ldflags "-X github.com/kitdot/nsh/cmd.nshVersion=$(git describe --tags --always)" -o nsh .
```

### macOS universal binary（可選）

```bash
CGO_ENABLED=1 GOARCH=arm64 go build -ldflags "-X github.com/kitdot/nsh/cmd.nshVersion=0.1.0" -o /tmp/nsh-arm64 .
CGO_ENABLED=1 GOARCH=amd64 CC="clang -arch x86_64" go build -ldflags "-X github.com/kitdot/nsh/cmd.nshVersion=0.1.0" -o /tmp/nsh-amd64 .
lipo -create /tmp/nsh-arm64 /tmp/nsh-amd64 -output nsh
```

## 快速上手

```bash
nsh                 # 開啟主 TUI（群組 + 釘選）
nsh c web1          # 直接連線
nsh n               # 新增 host
nsh e web1          # 編輯 host
nsh d web1          # 刪除 host
nsh p               # 只看釘選 host
nsh exp             # 匯出
nsh imp backup.nsh.enc
nsh -v              # 版本
```

`conn` / `copy` / `edit` / `del` / `auth` / `show` 沒給 alias 時，會進入互動式選擇器。

## 指令總覽

| Command | Alias | 說明 |
|---|---|---|
| `conn [alias]` | `c` | SSH 連線 |
| `pin` | `p` | 釘選 host 專用視圖 |
| `new` | `n` | 新增 host |
| `copy [alias]` | `cp` | 複製 host 為新項目 |
| `edit [alias]` | `e` | 編輯 host |
| `del [alias\|group]` | `d` | 刪除 host 或群組 |
| `order [scope]` | `o` | 調整群組、群組內 host、釘選順序（`scope`: `group` / `host` / `pinned`） |
| `auth [alias]` | `au` | 更新驗證方式 |
| `export` | `exp` | 匯出 host |
| `import [file]` | `imp` | 匯入 host |
| `list` | `l` | `nsh` 的別名（同主 TUI） |
| `show [alias]` | `s` | 顯示 host 詳細設定 |
| `config [key] [value]` | `conf` | 讀寫設定 |
| `completion` | — | 互動式安裝/移除 shell completion |
| `help` | `h` | 指令說明 |

## 常用流程

### 1) 新增 / 複製 / 編輯

- `nsh n`：逐步填寫 alias、HostName、User、Port、auth、群組、描述
- `nsh cp web1`：以既有 host 為模板建立新 host
- `nsh e web1`：修改既有 host；改 alias 時會同步處理 Keychain 密碼映射

### 2) 驗證方式管理

`nsh au web1` 可切換：

- None（不額外處理）
- Password（密碼存在 Keychain，連線時自動填入）
- Private key（連線前自動 `ssh-add --apple-use-keychain`）

### 3) 刪除 host / 群組

```bash
nsh d web1
nsh d web1 -y
nsh d Production --is-group
```

刪群組時可選：

- 只移除群組（host 轉到 `Uncategorized`）
- 刪除群組與全部 host（會再確認）

`Host *`（global default）受保護，不能刪除。

### 4) 排序

```bash
nsh o
nsh o group
nsh o host Dev
nsh o pinned
```

互動式拖曳順序：`Space` 抓取/放下、`↑↓` 移動、`Enter` 儲存、`Esc` 取消。

### 5) 匯出 / 匯入

`nsh exp`：

- Basic：輸出 `.nsh.json`（不含密碼與金鑰）
- Full：輸出 `.nsh.enc`（含密碼與金鑰，AES-256-GCM 加密）
- 多群組時可選擇「全部」或「指定群組」
- Full 匯出需要設定加密密碼並通過 Touch ID

`nsh imp [file]`：

- 支援 `.nsh.json` 與 `.nsh.enc`
- `.nsh.enc` 需要「Touch ID + 匯出密碼」
- 若純 JSON 仍含 secrets，一樣需要 Touch ID
- 衝突策略：逐筆詢問 / 全跳過 / 全覆蓋 / 全改名
- 密碼回寫 Keychain；金鑰寫入 `~/.ssh/nsh/`（`0600`）並更新 `IdentityFile`

## 主畫面快捷鍵

### Groups 視圖（`nsh` / `nsh l`）

| Key | 動作 |
|---|---|
| `Enter` | 連線或展開群組 |
| `e` | 編輯選中 host |
| `d` | 刪除選中 host |
| `n` | 新增 host |
| `p` | 釘選/取消釘選 |
| `Tab` | 切換到 Pinned 視圖 |
| `/` | 搜尋（空白 = AND） |
| `Esc` | 清除搜尋 / 收合 / 離開 |
| `↑↓` 或 `jk` | 導航 |

### Pinned 視圖（`nsh p` 或主畫面按 `Tab`）

| Key | 動作 |
|---|---|
| `Enter` | 連線 |
| `e` | 編輯 |
| `d` | 刪除 |
| `p` | 取消釘選 |
| `Space` | 進入重排模式 |
| `Tab` | 回到 Groups 視圖 |
| `/` | 搜尋 |
| `Esc` | 清除搜尋 / 離開 |

## Completion

```bash
nsh completion
```

互動安裝/移除支援 `zsh`、`bash`、`fish`（PowerShell 提供輸出腳本）。

手動輸出：

```bash
nsh completion zsh > ~/.zsh/completions/_nsh
nsh completion bash > ~/.local/share/bash-completion/completions/nsh
nsh completion fish > ~/.config/fish/completions/nsh.fish
```

## 設定（`nsh config`）

目前支援：

- `mode`: `auto`（預設）/ `fzf` / `list`

```bash
nsh conf
nsh conf mode
nsh conf mode fzf
```

## 全域旗標

```bash
nsh -v, --version
nsh --ssh-config <path>
```

`--ssh-config` 會改變主 SSH config 路徑，`nsh` 會對應使用 `<dir>/nsh/config` 作為受管檔案。

## 標籤協議（Tag Protocol）

`nsh` 在 `~/.ssh/nsh/config` 使用 `# nsh:` 標籤描述 metadata：

```ssh
# nsh: group=Production, desc=Main web server, auth=password, order=1
Host web1
    HostName 192.168.1.1
    User deploy
    Port 22
    IdentityFile ~/.ssh/prod_key
```

- `group`：群組名稱
- `desc`：描述
- `auth`：`password` 或 `key`
- `order`：群組內排序

另外：

- `# nsh-groups:` 管理群組顯示順序
- `# nsh-pinned:` 管理釘選順序

## 安全與資料完整性

- Lossless parser：保留原始格式、註解、`Include`、`Match`
- Atomic write：tmp -> rename
- 備份輪替：寫入前建立 `.nsh.bak`
- 權限控制：config 與金鑰檔案權限維持 `0600`
- Keychain：密碼不寫入 SSH config
- Touch ID：敏感匯入/匯出需要驗證

## 專案結構（精簡）

```text
nsh-go/
├── main.go
├── cmd/       # Cobra commands
├── bridge/    # Bubble Tea TUI components
├── core/      # parser, config manager, keychain, crypto
└── connect/   # ssh execution / PTY integration
```

## License

MIT
