# ローカル起動手順

本ドキュメントでは、開発者がローカル環境で gRPC サーバーを起動し、マイグレーション実行や API アクセスを確認するまでの手順を説明します。

## 前提条件
- Go 1.22 以上がインストールされていること
- Docker / Docker Compose が利用可能であること
- Buf CLI (`buf`) がインストール済みであること
- gRPC クライアントとして `grpcurl` もしくは `evans` 等を利用可能な状態であること（確認用）

> 補足: これらのツールは `README.md` の Quick Start でも推奨されています。

## 手順概要
1. 依存関係の取得
2. PostgreSQL コンテナの起動
3. データベースマイグレーションの適用
4. gRPC サーバーの起動
5. gRPC API へのアクセス確認
6. 後片付け

以下では各ステップを詳細に説明します。

### 1. 依存関係の取得
```bash
# Go モジュールの依存関係を同期
GO111MODULE=on go mod tidy

# プロトコル定義の lint とコード生成（必要に応じて）
(cd proto && buf lint && buf generate)
```

### 2. PostgreSQL コンテナの起動
```bash
# Docker Compose を利用して PostgreSQL を起動
docker compose --profile local up -d postgres

# 状態を確認（healthy になっていることを確認）
docker compose ps postgres
```

`docker compose up` を実行したディレクトリはリポジトリルート (`codex-grpc-clean-arch/`) にしてください。

### 3. データベースマイグレーションの適用
`assets/local.yaml` を利用する想定です。Docker Compose の `server` コンテナからは内部ネットワーク上の PostgreSQL に接続するため、`assets/local-compose.yaml` を使用します。環境変数 `CONFIG_PATH` を切り替えることでシナリオごとに設定を選択できます。

```bash
# 最新のマイグレーションを適用
CONFIG_PATH=assets/local.yaml go run ./cmd/migrate up

# 適用済みバージョンを確認（任意）
CONFIG_PATH=assets/local.yaml go run ./cmd/migrate version
```

Makefile を利用する場合は `make migrate-up` や `make migrate-version` でも同様の操作が可能です。

### 4. gRPC サーバーの起動
ローカルで Go を直接実行するか、Docker Compose + Air を利用したホットリロード運用を選べます。

```bash
# Go を直接実行する場合（Air を使わないシンプルな方法）
CONFIG_PATH=assets/local.yaml go run ./cmd/server
```

Air を利用したホットリロード手順は以下の通りです。初回は開発用イメージをビルドしてください。

```bash
# 1. 開発用コンテナのビルド（初回のみ）
docker compose --profile local build server

# 2. ホットリロード付きで起動（前面でログを確認する場合）
make dev-up

# 別ターミナルや停止時は以下で終了
make dev-down
```

`make` を使わない場合は `docker compose --profile local up server` / `docker compose --profile local down` で同等の操作が可能です。Air は `cmd/`, `internal/`, `pkg/`, `proto/` 以下の Go ファイル更新を監視し、再ビルド後にサーバーを自動的に再起動します。PostgreSQL と同時に起動する場合は `docker compose --profile local up` を利用すると `server` と `postgres` の両方が立ち上がります。

どの方法でもデフォルトでは `localhost:50051` で gRPC サーバーが待ち受けます。

### 5. gRPC API へのアクセス確認
`grpcurl` を使用して `GreeterService` の `SayHello` メソッドを呼び出す例です。

```bash
# TLS 無効のローカル環境向けサンプル
grpcurl -plaintext localhost:50051 greeter.v1.GreeterService.SayHello
```

レスポンスは現状 `{}` が返ります。サーバーは gRPC リフレクションを有効にしているため、`grpcurl` は proto ファイルを追加指定しなくても呼び出せます。ユーザーサービス等の詳細なインターフェースは `docs/api/` 配下を参照してください。

### 6. 後片付け
作業終了後は起動したコンテナを停止します。

```bash
# サービス停止とボリューム維持
docker compose down

# 永続ボリュームも含めて削除する場合（テスト用に DB を初期化したいとき）
docker compose down -v
```

## トラブルシュート
- **マイグレーションで接続エラーが発生する**: PostgreSQL コンテナの状態を `docker compose logs postgres` で確認し、`assets/local.yaml` の接続設定（ホスト: `localhost`, ポート: `15432` など）が合っているか確認してください。
- **ポート競合**: 既に `50051` や `15432` を使用しているプロセスがある場合、`assets/local.yaml` をコピーしてポートを変更し、`CONFIG_PATH` で切り替えてください。
- **gRPC クライアントが接続できない**: `server` の起動ログにエラーが出ていないか確認し、ファイアウォールや VPN によるローカルポート遮断がないか確認します。

## 参考
- `README.md` の Quick Start セクション
- `docs/api/` 内の各サービス仕様
- `Makefile` 内の `docker-up` / `migrate-up` などのターゲット
