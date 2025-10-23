# Architecture Overview

本リポジトリは Clean Architecture をベースに gRPC サーバーを構築することを目的としています。ここではレイヤー構成、主要コンポーネント、開発フローの注意点を整理します。

## Layering
- **Entities (`internal/core/*`)**: ドメインの基本的な構造体とビジネスルール。現在は `hello` に加え `user` ドメインを実装し、ユーザーエンティティ・値オブジェクト・ドメインエラーを保持します。
- **Use Cases (`internal/core/*`)**: 入出力ポート (インターフェース) を定義し、エンティティを操作するアプリケーションロジック。ユーザーユースケースでは `CreateUser`/`UpdateUser`/`DeleteUser` を提供し、リポジトリを介して永続化します。
- **Interface Adapters (`internal/adapters`)**: gRPC ハンドラ、DB リポジトリ、外部サービスクライアントなど。`internal/adapters/grpc/handler` には `GreeterHandler` と `UserGrpcHandler` を配置し、`internal/adapters/repository/postgres` に PostgreSQL 実装を置きます。
- **Framework & Drivers (`internal/platform`, `cmd/server`)**: 設定ロード、ロギング、依存性注入、アプリケーション起動。`internal/platform/server` で gRPC サーバーを組み立て、`cmd/server` で設定読み込み・DB 初期化・ユースケース注入を行います。

## gRPC Flow
1. gRPC サービス実装がリクエストを受け取り、DTO からユースケース入力モデルへ変換します。読み取り系か更新系かに応じて、アプリケーションサービスに対して Read Only / Read Write のトランザクション実行を指示します。
2. ユースケースがトランザクションマネージャ (`TransactionManager`) を介してビジネスルールを実行し、リポジトリへアクセスします。ユースケース内部では `WithinReadOnly`/`WithinReadWrite` を呼び分けてトランザクション境界を明示します。
3. 結果を DTO に戻し、レスポンスを生成します。エラーはドメインエラーとインフラエラーに分類し、`status.Status` へ適切に変換します。

## Configuration & Environment
- 設定ファイルは `assets/` 配下の YAML で管理し、`CONFIG_PATH` 環境変数（未指定時は `assets/local.yaml`）から読み込みます。
- Secrets や資格情報はローカル `.env` (コミット禁止) または Secret Manager 等で管理してください。
- ローカル検証は Docker Compose を用い、`postgres` サービス（PostgreSQL 16）と `server` サービスを起動します。`assets/migrations` には `golang-migrate` 形式のマイグレーションを配置しています。
- CI/CD では `buf lint`, `go test ./...`, `golangci-lint run` を段階的に実行するワークフローを用意します。将来的にはマイグレーション実行や統合テスト（`integration` ビルドタグ）も追加します。

## Testing Strategy
- ユースケースはテーブルドリブンテストで徹底的にカバーします。
- インフラ層はモックと integration テストを併用し、Docker Compose 等で周辺サービスを立ち上げる想定です。
- gRPC レイヤーは `grpc-go` のインプロセスサーバーか `buf` のエンドツーエンドテストツールで検証します。

## Next Steps
- マイグレーション実行用 CLI／Make ターゲットを追加し、CI でも `migrate up` を検証できるようにする。
- `internal/adapters/repository/postgres` を用いた統合テストを `test/` 配下に追加し、Docker 上の PostgreSQL で CRUD を検証する。
- 認証・監査ログ・ソフトデリートなど、ユーザードメインの拡張要件を整理し Issue 化する。
- gRPC エンドポイントの監視やメトリクス（OpenTelemetry など）を導入する方針を検討する。
