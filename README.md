# Discord Bot Server (Go) with LangChain

LangChain-Goを使用したDiscord chatbotです。Ollamaと連携して高性能な会話機能を提供します。

## セットアップ

### 1. 依存関係のインストール
```bash
go mod tidy
```

### 2. 環境設定
`.env.example`を`.env`にコピーして、設定を行ってください。

```bash
cp .env.example .env
```

`.env`ファイルを編集：
```
DISCORD_BOT_TOKEN=your_actual_bot_token_here
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama2
```

### 3. Discord Bot の作成
1. [Discord Developer Portal](https://discord.com/developers/applications)にアクセス
2. 新しいアプリケーションを作成
3. "Bot"セクションでbotを作成
4. **Message Content Intent**を有効化
5. Botトークンをコピーして`.env`ファイルに設定

### 4. Ollamaの起動
```bash
ollama serve
ollama pull llama2
```

### 5. 実行
```bash
go run cmd/bot/main.go
```

## 機能

- **AI会話機能** - Ollamaを使用した自然な会話
- **チャンネル別履歴管理** - LangChainによる効率的なメモリ管理
- **基本コマンド**:
  - `!ping` - "Pong!"と応答
  - `!hello` - ユーザー名付きで挨拶
  - `!clear` - 会話履歴をクリア

## 技術スタック

- **Go 1.24**
- **LangChain-Go** - 会話履歴とLLM管理
- **Ollama** - ローカルLLMプロバイダー
- **DiscordGo** - Discord API

## プロジェクト構造

```
├── cmd/bot/          # メインアプリケーション
├── pkg/bot/          # LangChain統合ロジック
├── .env.example      # 環境変数の例
└── README.md         # このファイル
```