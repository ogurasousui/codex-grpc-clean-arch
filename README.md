# codex-grpc-clean-arch

Go 言語と Clean Architecture を採用した gRPC サーバーのテンプレートです。プロジェクト全体で責務を分離し、疎結合なユースケースとアダプタを構築するためのガイドラインとサンプル構成を提供します。

## Quick Start
- **Prerequisites**: Go 1.22+, Buf CLI または Docker, protoc, `make` (任意)、Docker。
- **依存関係の初期化**: `go mod init github.com/ogurasousui/codex-grpc-clean-arch` を実行し、必要なライブラリ (`google.golang.org/grpc` など) を `go get` で追加します。
- **プロトコル定義の検証/生成**: `docker run --rm -v $PWD:/workspace -w /workspace bufbuild/buf lint` / `... generate` を利用するとローカルに Buf をインストールせずに済みます。`buf.yaml` と `buf.gen.yaml` は `proto/` 直下に配置してください。
- **ローカル動作確認**: `docker compose up server` で gRPC サーバーを起動します（Go 1.24 ベースの `golang:1.24-bullseye` イメージを使用）。テストは `docker compose run --rm server go test ./...` で実行できます。直接実行する場合は `CONFIG_PATH=assets/local.yaml go run ./cmd/server` を利用します。

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
