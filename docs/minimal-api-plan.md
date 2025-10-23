# シンプル API 実装計画

## 1. プロトコル定義
- `proto/service.proto` を作成し、以下の要件を満たす gRPC サービスを定義する。
  - サービス名: `GreeterService`（暫定）。
  - メソッド: `rpc SayHello(SayHelloRequest) returns (SayHelloResponse);`。
  - リクエストメッセージ `SayHelloRequest` は現状フィールド無しのプレースホルダー。
  - レスポンスメッセージ `SayHelloResponse` に `string message = 1;` を持たせ、初期実装では空文字列を返す。
- `buf.yaml` / `buf.gen.yaml` を `proto/` 直下に追加し、`buf generate` で Go コードを生成できるようにする。

## 2. コード生成とディレクトリ準備
- `buf mod init` を実行し、Go 用の `go_package` を設定する。
- 生成物は `internal/adapters/grpc/gen` など明示的な出力先へ配置し、手書きコードと分離する。
- `go.mod` を初期化し、`google.golang.org/grpc` など必要な依存を追加する。

## 3. Clean Architecture スキャフォールディング
- `internal/core/hello` にユースケースインターフェース `type Greeter interface { SayHello(ctx context.Context) (string, error) }` を配置。
- 同パッケージにドメインサービスの実装（空文字列を返すスタブ）とユニットテストを追加。
- `internal/adapters/grpc/handler` に gRPC ハンドラを実装し、生成コードのインターフェースを満たす形でユースケースを呼び出す。
- `internal/platform/server` にサーバー起動ロジック、DI、ロギングを集約する。

## 4. エントリーポイントと動作確認
- `cmd/server/main.go` で設定読込（暫定でハードコードでも可）と gRPC サーバー起動処理を記述。
- ローカル検証は Docker Compose を用い、`docker compose up server` で gRPC サーバーを起動する。イメージは Go 1.25.3 系 (`golang:1.25.3-bookworm`) を利用する。
- `docker compose run --rm server go test ./...` を実行し、コンテナ上でユニットテストが通ることを確認。

## 5. API ドキュメント整備
- `docs/api/` 配下に gRPC メソッド仕様を記載したドキュメントを追加する（例: `docs/api/greeter.md`）。
- ドキュメントにはリクエスト/レスポンスの型、サンプル、想定する認証要件（現時点では無し）を明記する。
- Buf の `buf lint --path proto` や `buf breaking` などのコマンドを併記し、API 変更時のチェック手順を示す。

## 6. ドキュメントと今後の拡張
- `README.md` に gRPC エンドポイント概要と Docker での動作確認手順を追記する。
- `docs/architecture-overview.md` に gRPC レイヤーとユースケースの関連を更新する。
- GitHub Actions (`.github/workflows/ci.yml`) で `go test ./...` を実行するパイプラインを維持し、追加チェックが必要になればジョブを拡張する。

## 7. Git フローと PR 作成
- `feat/simple-greeter` などの feature ブランチを `main` から切り出す (`git checkout -b feat/simple-greeter`)。
- 実装・テスト・ドキュメント変更をコミットし、`git push origin feat/simple-greeter` でリモートへプッシュする。
- GitHub 上で Pull Request を作成し、`AGENTS.md` のガイドラインに沿って説明、検証手順、関連 Issue を記載する。
