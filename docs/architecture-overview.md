# Architecture Overview

本リポジトリは Clean Architecture をベースに gRPC サーバーを構築することを目的としています。ここではレイヤー構成、主要コンポーネント、開発フローの注意点を整理します。

## Layering
- **Entities (`internal/core/entity`)**: ドメインの基本的な構造体とビジネスルール。外部依存を持たない純粋な Go コードに限定します。
- **Use Cases (`internal/core/usecase`)**: 入出力ポート (インターフェース) を定義し、エンティティを操作するアプリケーションロジック。DB・ネットワーク層へはポート経由で依存します。
- **Interface Adapters (`internal/adapters`)**: gRPC ハンドラ、DB リポジトリ、外部サービスクライアントなど。ユースケースが定義するポートに実装を提供します。
- **Framework & Drivers (`internal/platform`, `cmd/server`)**: 設定ロード、ロギング、依存性注入、アプリケーション起動。`internal/platform/server` で gRPC サーバーを組み立て、`cmd/server` でエントリーポイントを提供します。

## gRPC Flow
1. gRPC サービス実装がリクエストを受け取り、DTO からユースケース入力モデルへ変換します。
2. ユースケースがバリデーションとビジネスルールを実行し、出力ポート (リポジトリなど) を介して永続化・外部連携を行います。
3. 結果を DTO に戻し、レスポンスを生成します。エラーはドメインエラーとインフラエラーに分類し、`status.Status` へ適切に変換します。

## Configuration & Environment
- 設定ファイルは `assets/` に YAML で配置し、`CONFIG_PATH` 環境変数で読み込む想定です。
- Secrets や資格情報はローカル `.env` (コミット禁止) または Secret Manager 等で管理してください。
- ローカル検証は Docker コンテナで行います。`docker compose` によるマルチサービス構成を前提にしつつ、初期段階では単一コンテナで `go test ./...` や `go run ./cmd/server` を実行します。
- CI/CD では `buf lint`, `go test ./...`, `golangci-lint run` を段階的に実行するワークフローを用意します。インフラ本番デプロイは将来的な課題として後回しにします。

## Testing Strategy
- ユースケースはテーブルドリブンテストで徹底的にカバーします。
- インフラ層はモックと integration テストを併用し、Docker Compose 等で周辺サービスを立ち上げる想定です。
- gRPC レイヤーは `grpc-go` のインプロセスサーバーか `buf` のエンドツーエンドテストツールで検証します。

## Next Steps
- 最低限の scaffolding (module 初期化、`cmd/server/main.go`) を作成し、CI を GitHub Actions で立ち上げる。
- プロトコル定義の MVP を `proto/` に追加し、コード生成を自動化する。
- ドメインユースケースと DB アダプタをスパイクし、依存方向を検証する。
- 現状は手動生成した gRPC スタブを使用しているため、Buf/`protoc` で再生成し置き換える。
