# TCP Socket Chat

GoでTDDを実践して作成したTCP socketベースのチャットアプリケーションです。

## 概要

このプロジェクトは、TCP socketを使用したシンプルなチャットシステムです。サーバーとクライアントの2つのCLIツールが含まれており、複数のユーザーがリアルタイムでメッセージをやり取りできます。

## 機能

- **TCP Socket通信**: 生のTCPソケットを使用した通信
- **複数クライアント対応**: 同時に複数のユーザーが接続可能
- **メッセージブロードキャスト**: 1人のメッセージが他の全員に配信される
- **Join/Leave通知**: ユーザーの入退室が通知される
- **並行処理**: Goroutineを使った効率的な並行処理

## 必要な環境

- Go 1.25.2以上

## インストール

### 1. リポジトリをクローン

```bash
git clone <repository-url>
cd tcp-socket
```

### 2. ビルド

サーバーとクライアントのバイナリをビルドします：

```bash
# サーバーをビルド
go build -o bin/server ./cmd/server

# クライアントをビルド
go build -o bin/client ./cmd/client
```

## 使い方

### サーバーの起動

まず、サーバーを起動します：

```bash
./bin/server -port :8080
```

オプション：
- `-port`: サーバーが待ち受けるポート（デフォルト: `:8080`）

サーバーが起動すると、以下のようなメッセージが表示されます：
```
Starting server on :8080...
Server started on [::]:8080
```

### クライアントの接続

別のターミナルでクライアントを起動します：

```bash
./bin/client -server localhost:8080 -username alice
```

オプション：
- `-server`: 接続先サーバーのアドレス（デフォルト: `localhost:8080`）
- `-username`: チャットで表示されるユーザー名（必須）

クライアントが接続すると、以下のようなメッセージが表示されます：
```
Connected to localhost:8080 as alice
Type your messages (or 'quit' to exit):
```

### チャットの使い方

1. メッセージを入力してEnterキーを押すと、接続している他の全てのユーザーに送信されます
2. 他のユーザーからのメッセージは `[username]: message` の形式で表示されます
3. ユーザーの入退室は `*** username joined the chat ***` の形式で通知されます
4. 終了する場合は `quit` または `exit` と入力します

## 使用例

### 3人でチャットする場合

**ターミナル1: サーバー起動**
```bash
$ ./bin/server -port :8080
Starting server on :8080...
Server started on [::]:8080
User alice joined
User bob joined
Message from alice: Hello everyone!
User carol joined
Message from bob: Hi alice!
Message from carol: Hey guys!
```

**ターミナル2: alice として接続**
```bash
$ ./bin/client -server localhost:8080 -username alice
Connected to localhost:8080 as alice
Type your messages (or 'quit' to exit):
*** bob joined the chat ***
Hello everyone!
[bob]: Hi alice!
*** carol joined the chat ***
[carol]: Hey guys!
```

**ターミナル3: bob として接続**
```bash
$ ./bin/client -server localhost:8080 -username bob
Connected to localhost:8080 as bob
Type your messages (or 'quit' to exit):
[alice]: Hello everyone!
Hi alice!
*** carol joined the chat ***
[carol]: Hey guys!
```

**ターミナル4: carol として接続**
```bash
$ ./bin/client -server localhost:8080 -username carol
Connected to localhost:8080 as carol
Type your messages (or 'quit' to exit):
[alice]: Hello everyone!
[bob]: Hi alice!
Hey guys!
```

## テストの実行

プロジェクトには包括的なテストが含まれています：

```bash
# 全てのテストを実行
go test ./...

# 詳細な出力付きで実行
go test ./... -v

# 特定のパッケージのテストを実行
go test ./pkg/protocol -v
go test ./internal/server -v
go test ./internal/client -v
go test ./test -v
```

## プロジェクト構成

```
tcp-socket/
├── cmd/
│   ├── server/          # サーバーCLIツール
│   │   └── main.go
│   └── client/          # クライアントCLIツール
│       └── main.go
├── internal/
│   ├── server/          # サーバー実装
│   │   ├── server.go
│   │   └── server_test.go
│   └── client/          # クライアント実装
│       ├── client.go
│       └── client_test.go
├── pkg/
│   └── protocol/        # メッセージプロトコル
│       ├── message.go
│       └── message_test.go
├── test/
│   └── integration_test.go  # 統合テスト
├── go.mod
└── README.md
```

## 技術詳細

### メッセージプロトコル

メッセージは以下の3種類：
- **TEXT**: 通常のチャットメッセージ
- **JOIN**: ユーザーが参加した通知
- **LEAVE**: ユーザーが退出した通知

メッセージのエンコード/デコードにはGo標準の`encoding/gob`を使用しています。

### 並行処理

- サーバーは各クライアントを別々のgoroutineで処理
- クライアントはメッセージ受信を別のgoroutineで処理
- 全ての共有リソースはmutexで保護されており、スレッドセーフ

## TDDについて

このプロジェクトはTest-Driven Development (TDD)の手法で開発されました：

1. **Red**: テストを先に書く（失敗するテスト）
2. **Green**: テストを通過させる最小限の実装
3. **Refactor**: コードを改善

コミット履歴を見ると、各機能がRed-Greenサイクルで開発されていることが分かります。

## トラブルシューティング

### ポートが既に使用されている

```
Failed to start server: listen tcp :8080: bind: address already in use
```

別のポート番号を指定してください：
```bash
./bin/server -port :9090
```

### サーバーに接続できない

- サーバーが起動しているか確認してください
- ファイアウォールがポートをブロックしていないか確認してください
- サーバーアドレスとポート番号が正しいか確認してください

## ライセンス

This project is open source and available under the MIT License.
