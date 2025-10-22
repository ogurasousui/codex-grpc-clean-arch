# Repository Guidelines

## プロジェクト構成とモジュール配置
本リポジトリは Clean Architecture に基づく Go 製 gRPC サーバーを対象としています。以下の構成方針に沿ってコードを配置してください。`proto/` には protobuf 契約と Buf 設定、`cmd/server/` には実行エントリーポイント、`internal/core/` にはエンティティ・値オブジェクト・ユースケース（外部依存禁止）、`internal/adapters/` には gRPC ハンドラやリポジトリなど入出力アダプタ、`internal/platform/` には設定読み込み・ロギング・DB クライアントなどのインフラ層、`pkg/` には再利用可能なユーティリティ、`assets/` にはサンプル設定やマイグレーション素材、`test/` には統合テスト群を置きます。必要なディレクトリは実装開始時に順次作成してください。

## ビルド・テスト・開発コマンド
Go 1.22 以上と Buf をインストールしてください。`buf lint` で protobuf のスタイルを検証し、契約変更後は `buf generate` または `go generate ./...` でスタブを再生成します。ユニット/統合テストは Docker コンテナ上での実行を基本とし、`docker run --rm -v $PWD:/app -w /app golang:1.22-bullseye go test ./...` のように実行します。ローカル起動も Docker Compose を想定し、準備が整うまでは `CONFIG_PATH=assets/local.yaml go run ./cmd/server` を用いて動作を確認して構いません。ホットリロードを導入する場合は `air` や `reflex` を利用し、追加ツールはドキュメントに追記してください。

## コーディングスタイルと命名規約
コミット前に必ず `gofmt` もしくは `goimports` を適用します。Makefile を導入した際は `make format` でフォーマットを一括実行できるようにします。関数名は lowerCamelCase、公開構造体は UpperCamelCase、protobuf フィールドは snake_case を基本とします。`internal/core` のインターフェースはドメイン表現を意識し、アダプタ実装には役割を示すサフィックス（例: `UserRepository`, `UserGrpcHandler`）を付けます。`golangci-lint run` で lint を回し、必要に応じて `.golangci.yml` にルールを追加してください。

## テストガイドライン
Go 標準の `testing` パッケージと `testify` を用いてテストを作成します。各パッケージのテストは同一ディレクトリに配置し、ファイル名は `*_test.go` とします。テスト関数は振る舞いが分かるよう `TestCreateUser_Valid` のような命名を推奨します。統合テストは `test/` 配下にまとめ、長時間実行されるケースは `integration` ビルドタグで切り替えられるようにしてください。`internal/core` は 80% 以上のカバレッジを目標とし、未達の場合は PR 説明に理由を記載します。

## コミット・プルリクエスト運用
自動リリース連携のため Conventional Commits（`feat:`, `fix:`, `chore:` など）を採用します。プルリクエストでは関連 Issue、意思決定の背景、検証手順を明記してください。影響したレイヤー（`core`, `adapters`, `platform` など）に応じて適切なレビュワーを指定し、proto のスキーマを変更した場合は生成物を含めて更新します。
- 作業開始前に `main` ブランチから `feat/<feature-name>` 形式のフィーチャーブランチを作成し、作業は常に専用ブランチ上で実施してください。
- 行動計画や設計内容が作業者・関係者間で承認されたタイミングでフィーチャーブランチから PR を作成し、レビュー依頼と検証手順を提示してください。

## コミュニケーション
- Issues、プルリクエスト、ドキュメントでのやりとりは日本語を基本としてください。英語資料を引用する場合は要点の日本語要約も添えてください。
- CLI 上のエージェントとの対話も原則日本語で行い、必要に応じて補足として英語表現を併記してください。
- GitHub 上の操作（PR 作成、レビュー依頼など）は MCP 経由の GitHub ツールを利用して行ってください。

## その他
- 日本語で簡潔かつ丁寧に回答してください
