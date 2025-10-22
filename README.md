# codex-grpc-clean-arch

Go 言語と Clean Architecture を採用した gRPC サーバーのテンプレートです。プロジェクト全体で責務を分離し、疎結合なユースケースとアダプタを構築するためのガイドラインとサンプル構成を提供します。

## Quick Start
- **Prerequisites**: Go 1.22+, Buf CLI（または `bufbuild/buf` Docker イメージ）、Docker & Docker Compose、`golang-migrate`（マイグレーション実行用に推奨）。
- **依存関係の同期**: `go mod tidy` を実行し、プロジェクトで利用するライブラリ（`pgx`, `yaml`, `testify` など）を取得します。
- **プロトコル定義の検証/生成**: `cd proto && buf lint` / `buf generate` を実行します。Docker を使う場合は `docker run --rm -v $PWD:/workspace -w /workspace bufbuild/buf generate` のように呼び出します。
- **PostgreSQL の起動**: `docker compose up -d postgres` で開発用 DB を立ち上げます。
- **マイグレーション**: `go run ./cmd/migrate up` で `assets/migrations` を適用できます（`down`, `drop`, `version` もサポート）。外部ツール `golang-migrate` を使う場合は同ディレクトリを参照してください。
- **シードデータ**: 統合テスト等で初期データが必要な場合は `go run ./cmd/migrate -dir assets/seeds up` を実行します（`down` で巻き戻し可能）。
- **サーバーの起動**: `CONFIG_PATH=assets/local.yaml go run ./cmd/server` もしくは `docker compose up server` で gRPC サーバーを起動します。
- **テスト実行**: `go test ./...` または `docker compose run --rm server go test ./...` でユニットテストを実行します。PostgreSQL を使用する統合テストは `CONFIG_PATH=assets/local.yaml go test -tags=integration ./test/...` で実行します。

## Project Layout
```
cmd/server/         エントリーポイントと DI 初期化
internal/core/      エンティティ・ユースケースなど純粋なビジネスロジック
internal/adapters/  gRPC ハンドラ、DB・外部 API との接続実装
internal/platform/  ロギング、設定、DB クライアントなどのプラットフォーム層
pkg/                他プロジェクトでも再利用可能なユーティリティ
proto/              protobuf 定義と Buf 設定
assets/             設定ファイルやマイグレーション素材
test/               統合テスト、エンドツーエンドシナリオ
docs/               追加ドキュメント (設計・運用ガイドなど)
```

## Development Workflow
1. ユースケースを `internal/core` に追加し、インターフェースを定義します。
2. `internal/adapters` で入出力層 (gRPC、DB) を実装し、ユースケースを注入します。
3. `internal/platform` で設定やライフサイクルを管理し、`cmd/server/main.go` で組み合わせます。
4. `docker compose run --rm server go test ./...` でユニットテストを確認し、`-tags=integration` 付きで統合テストを実行します（ホストに Go がインストールされていなくても Docker で完結）。
5. PR 作成前に `buf lint` や `golangci-lint run` を通し、`AGENTS.md` のガイドラインに従って説明文・検証手順を記載します。
6. GitHub Actions (`.github/workflows/ci.yml`) が `go test ./...` を実行するため、テストが通る状態で push/PR を行ってください。

## Communication
Issue や Pull Request、ドキュメントでの議論は日本語を基本とします。英語資料を参照する場合は日本語で要点をまとめ、チーム内共有を円滑にしてください。

より詳細なアーキテクチャ指針や開発手順は `docs/` ディレクトリの資料を参照してください。
