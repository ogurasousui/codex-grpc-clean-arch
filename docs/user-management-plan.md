# ユーザー管理 API 実装計画

## 1. 現状把握と要件整理
- 既存の `GreeterService` 以外にユーザー機能は未実装であることを確認済み。
- 新たに gRPC ベースでユーザーの作成・更新・削除を提供し、PostgreSQL をデータストアとして採用する。
- Clean Architecture の各レイヤーに沿って責務を分割し、テストとドキュメントを整備する。
- 実装作業は `main` ブランチから `feat/user-management` などのフィーチャーブランチを切って進め、完了後にプルリクエストを作成する。
- 行動計画の内容について作業者が承認した段階でフィーチャーブランチから PR を作成し、レビューを依頼する。

## 2. インフラ & 設定整備
- `docker-compose.yaml` に PostgreSQL 16 サービスを追加し、初期化 SQL 用のボリュームをマウントする。
- `assets/local.yaml` を新規作成し、`server` と `database` セクションを定義（例: `host`, `port`, `user`, `password`, `dbname`, `sslmode`）。
- `internal/platform/config` パッケージを追加して YAML 設定読み込み・バリデーションを実装し、`cmd/server/main.go` から利用する。
- `internal/platform/db/postgres` を作成し、`pgxpool` を用いた接続管理とコンテキスト対応のヘルパーを用意する。
- ローカル開発では `docker compose up postgres` → `CONFIG_PATH=assets/local.yaml go run ./cmd/server` で起動できるようにする。

## 3. スキーマ & マイグレーション方針
- `assets/migrations` を作成し、`0001_create_users.sql` で `users` テーブル（`id`, `email`, `name`, `status`, `created_at`, `updated_at` など）を定義。
- 将来的な変更に備えて `golang-migrate/migrate` を採用し、`cmd/migrate/main.go` 等で CLI を用意するか Makefile からコマンド実行できるよう設計。
- テスト用に初期データ投入が必要であれば `assets/seeds/0001_seed_users.sql` を適用し（`go run ./cmd/migrate -dir assets/seeds up`）、`integration` タグ付きテストで活用する。

## 4. ドメイン (internal/core) 整備
- `internal/core/user` パッケージを新設し、エンティティ (`User`) と値オブジェクト（例: `UserID`, `Email`）を定義。
- リポジトリインターフェース `UserRepository`（`Create`, `Update`, `Delete`, `Get` など）を用意。
- ユースケース:
  - `CreateUser`：入力バリデーション、重複チェック、リポジトリ呼び出し。
  - `UpdateUserProfile`：存在確認、属性更新、監査フィールド更新。
  - `DeleteUser`：ソフトデリート or 物理削除の方針を決定（初期は物理削除を想定）。
- 各ユースケースのユニットテストを `testing` + `testify` で作成し、インメモリ実装のフェイクリポジトリを使用。

## 5. アダプタ整備
- `proto/service.proto` に `UserService` を追加し、以下の RPC を定義。
  - `CreateUser(CreateUserRequest) returns (CreateUserResponse)`
  - `UpdateUser(UpdateUserRequest) returns (google.protobuf.Empty)`
  - `DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty)`
- `buf generate` により `internal/adapters/grpc/gen/user/v1` 配下へコードを生成するよう `buf.gen.yaml` を更新。
- `internal/adapters/grpc/handler/user.go` を実装し、ユースケースを注入してトランスレーションを行う。エラーハンドリングは gRPC ステータスコードにマッピング。
- `internal/adapters/repository/postgres/user_repository.go` を作成し、SQL (`sqlc` もしくは `pgx` のプリペアドステートメント) でリポジトリインターフェースを実装。
- トランザクション境界の扱いを整理し、必要ならユースケース層で `UnitOfWork` インターフェースを導入する。

## 6. プラットフォーム・エントリーポイント更新
- `cmd/server/main.go` を更新して設定読み込み、DB プール初期化、ユースケースとアダプタの DI を実装。
- シャットダウン処理に DB クローズも追加し、GracefulStop と合わせて停止する。
- 必要に応じてロギング (`internal/platform/log`) を整備し、SQL 実行時のログ出力やリクエストトレースを準備。

## 7. テスト & 品質保証
- コア層のユニットテストで 80% 以上のカバレッジを維持。
- アダプタのインテグレーションテストを `test/` 配下に追加し、Docker 上の PostgreSQL を利用して CRUD の E2E を検証。`integration` ビルドタグで切り替え。
- CI（将来的には GitHub Actions）で `go test ./...`, `golangci-lint run`, `buf lint`, `docker compose run --rm migrate up` 等を実行するワークフローを設計。
- CI（GitHub Actions）で `go run ./cmd/migrate up`、`go test -tags=integration ./test/...` を実行し、PostgreSQL サービスを立ち上げて統合テストまで自動化する。

## 8. ドキュメント & 運用整備
- `docs/api/user-service.md` を追加し、RPC ごとのリクエスト/レスポンス例と gRPCurl コマンド例を記載。
- `README.md` に PostgreSQL サービスの起動手順とマイグレーション実行手順を追記。
- `docs/architecture-overview.md` にユーザー用ユースケースとリポジトリの流れを追記し、Clean Architecture の層ごとの責務を明示。

## 9. 未決事項・フォローアップ
- 認証/認可の要件が発生した際のメタデータ設計（トークン、監査ログ）。
- ユーザーの重複判定ロジック（メールアドレスユニーク制約でカバーか、別途仕様を持つか）。
- ソフトデリートを導入する場合のテーブル設計 (`deleted_at`) と検索条件の統一。
- 本番環境向けのマイグレーション実行フロー（例: CI/CD ステップ、手動承認）を後続で詰める。
