# ユーザー取得 API 拡張行動計画

## 1. 背景と目的
- 既存の `UserService` は作成・更新・削除のみを提供しており、単体取得および一覧取得の RPC が未実装。
- 管理画面や連携バッチから利用可能な読み取り系 API を追加し、ユーザーデータを参照できるようにする。
- Clean Architecture 構成を維持しつつ、ドメイン層からアダプタ層までの責務を整理して実装する。

## 2. 現状整理
- `proto/user/v1/user.proto` に `GetUser`/`ListUsers` に相当するメッセージ・RPC 定義が無い。
- `internal/core/user` の `UseCase` は `CreateUser`/`UpdateUser`/`DeleteUser` のみを公開し、`Repository` も `FindByID`・`FindByEmail` のみで一覧用メソッドが未定義。
- PostgreSQL 用リポジトリ実装 (`internal/adapters/repository/postgres/user_repository.go`) では単体取得までは可能だが、複数件取得ロジックが存在しない。
- gRPC ハンドラ (`internal/adapters/grpc/handler/user.go`) とテストは現行 RPC のみをカバーしている。

## 3. 対象範囲
- 対象: gRPC インタフェース、ドメインサービス、リポジトリ実装、ハンドラおよび関連テストの拡張。
- 対象外: 認証・認可、外部公開向け API Gateway、クライアント実装、管理画面 UI などの周辺機能。

## 4. 実装タスク
### 4.1 プロトコル定義
- `proto/user/v1/user.proto` に以下を追加。
  - `GetUserRequest` / `GetUserResponse`（主キー ID で取得）。
  - `ListUsersRequest`（`page_size`、`page_token`、`status` フィルタを持たせる）と `ListUsersResponse`（`repeated User users` と `next_page_token`）。
- `buf lint` でスタイルを検証し、`buf generate`（もしくは `go generate ./...`）で gRPC/Go スタブを再生成。生成物をコミット対象に含める。

### 4.2 ドメイン層 (`internal/core/user`)
- `Repository` インタフェースに `List(ctx context.Context, filter ListUsersFilter) ([]*User, string, error)` を追加し、ページネーション情報を返す設計にする。
- `ListUsersFilter` 構造体を新設し、`Limit`（最大件数）、`PageToken`（オフセット表現）、`Status`（アクティブ状態フィルタ）を保持。
- `UseCase` インタフェース／`Service` に `GetUser(ctx context.Context, in GetUserInput)` と `ListUsers(ctx context.Context, in ListUsersInput)` を追加。
- 既存の `fakeRepo`・`service_test.go` を拡張し、新しいユースケースの正常系・異常系（無効 ID、存在しないユーザー、ページサイズ超過など）をテストでカバー。80% 以上のカバレッジを維持する。

### 4.3 アダプタ層
- gRPC ハンドラ (`internal/adapters/grpc/handler/user.go`) に `GetUser` / `ListUsers` 実装を追加。`ListUsers` は gRPC リクエストの `page_size` をドメインの `Limit` にトリムし、レスポンスに `next_page_token` を付与。
- エラーマッピングを再利用しつつ、`GetUser` の `ErrUserNotFound` を `NotFound` に変換、`page_size` の妥当性エラーは `InvalidArgument` を返す。
- ハンドラ単体テストを追加し、`stubUserUseCase` に新メソッドと検証ロジックを実装。`ListUsers` の変換・エラー処理・ページング token の受け渡しを確認。
- PostgreSQL リポジトリ (`internal/adapters/repository/postgres/user_repository.go`) に一覧取得クエリを追加。作成日時降順でソートし、`LIMIT` と `OFFSET` を使用してページング。`next_page_token` は簡易にオフセット+取得件数を文字列化して返す。
- リポジトリのユニットテストを更新し、複数件取得とページングの境界条件（データが無い場合の空配列、最終ページでの `next_page_token` 空文字など）を検証。

### 4.4 プラットフォーム・DI
- `internal/platform/server/server.go` はハンドラ登録のみのため、追加 RPC を利用するためのコード変更は不要だが、コンパイルエラー解消のため新メソッド対応を確認。
- `cmd/server/main.go` での依存解決は既存のまま利用できるが、`user.UseCase` の新メソッド追加に伴うコンパイルエラーを解消する（`NewService` 呼び出しはそのままで可）。

## 5. テスト計画
- ドメイン層: `internal/core/user/service_test.go` に `TestService_GetUser_*`、`TestService_ListUsers_*` を追加。
- ハンドラ層: `internal/adapters/grpc/handler/user_test.go` に取得系 RPC のテストケースを追加し、gRPC ステータスコード変換を確認。
- リポジトリ層: `internal/adapters/repository/postgres/user_repository_test.go` に一覧取得用テストを新設。`pgxpool` をモック／テスト専用 DB（現在のテスト方針に合わせて `pgxpoolmock` または `dockertest`）で実行。
- 結合テスト: 余力があれば `test/` 配下に gRPC 経由で取得系を呼び出す統合テストを追加し、`integration` タグで管理。

## 6. ドキュメント・生成物更新
- `docs/api/user-service.md`（未作成の場合は新規）へ `GetUser` / `ListUsers` のリクエスト・レスポンス例、`grpcurl` コマンド例を記載。
- `README.md` に新規 RPC の簡単な利用例を追記し、`buf generate` の実行フローに言及。
- 必要に応じて API 変更点を CHANGELOG またはリリースノートに記載（運用方針と相談）。

## 7. リスクと懸念事項
- ページング方式: シンプルな offset ベース token は大規模データで性能劣化の懸念があるため、将来的にカーソル方式へ移行する余地を残すことをドキュメント化。
- ステータスフィルタ: インデックス無しでの検索による性能問題があり得るため、必要に応じて `status` カラムへインデックス追加を検討。
- API 互換性: 今後フィールド追加を考慮し、`ListUsersRequest` のデフォルト `page_size` を 50 件程度に制限し DoS を防止。

## 8. ブランチ戦略と進め方
- `main` から `feat/user-query-api` ブランチを作成し、本計画に沿って作業する。
- プロトコル変更→ドメイン層→リポジトリ→ハンドラ→テスト→ドキュメントの順でコミットを積むことでレビューを容易にする。
- 行動計画の内容に合意後、段階的に PR を作成し、レビューではスキーマ・ページング仕様・性能面に焦点を当てる。
