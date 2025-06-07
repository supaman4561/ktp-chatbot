# Discord Bot Server (Go)

Discord botサーバーのベース実装です。

## セットアップ

### 1. 依存関係のインストール
```bash
go mod tidy
```

### 2. 環境設定
`.env.example`を`.env`にコピーして、Discord botトークンを設定してください。

```bash
cp .env.example .env
```

`.env`ファイルを編集して、実際のDiscord botトークンを設定：
```
DISCORD_BOT_TOKEN=your_actual_bot_token_here
```

### 3. Discord Bot の作成
1. [Discord Developer Portal](https://discord.com/developers/applications)にアクセス
2. 新しいアプリケーションを作成
3. "Bot"セクションでbotを作成
4. Botトークンをコピーして`.env`ファイルに設定

### 4. 実行
```bash
go run cmd/bot/main.go
```

## 機能

現在実装されている基本機能：
- `!ping` - "Pong!"と応答
- `!hello` - ユーザー名付きで挨拶

## プロジェクト構造

```
├── cmd/bot/          # メインアプリケーション
├── pkg/bot/          # 公開パッケージ
├── internal/config/  # 内部設定
├── .env.example      # 環境変数の例
└── README.md         # このファイル
```