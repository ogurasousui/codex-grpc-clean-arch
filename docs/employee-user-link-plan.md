# employeesテーブルのユーザー紐付け行動計画

## 1. 背景と目的
- 社員個人情報（氏名・メールアドレスなど）を `employees` テーブルに保持しているが、今後は認証基盤や他ドメインと共通利用する `users` テーブルへ集約したい。
- `employees` が `users.id` を参照する構造へ変更し、個人属性の単一ソース化と更新整合性の向上を図る。
- 既存社員データの損失を防ぎながら段階的に移行し、API 利用者へも互換性方針を明示する。

## 2. 現状整理
- DB: `assets/migrations/0003_create_employees.up.sql` で `email`, `last_name`, `first_name` など個人属性を `employees` が直接保持。
- ドメイン: `internal/core/employee/employee.go` が同属性をエンティティに持ち、ユースケース・リポジトリも同前提。
- gRPC: `proto/employee/v1/employee.proto` の `Employee` メッセージおよび CRUD RPC が `email`, `last_name`, `first_name` を扱う。
- `users` ドメインは `internal/core/user`／`proto/user/v1` で既存実装済み。`users` テーブルには `email`, `name`, `status` が存在。

## 3. スコープと前提
- 対象: DB マイグレーション、`internal/core/employee` エンティティ・ユースケース、PostgreSQL リポジトリ、gRPC/proto、テスト、ドキュメント。
- 既存 `users` データとの重複回避が必須。`email` で一意識別を試み、未登録の場合は自動生成ポリシーを定義する。
- バックフィル時に `email` が `NULL` の社員は一旦プレースホルダーを作成し、後続で要修正項目としてレポートする（要件レビューで確定）。
- API 互換性: フィールド削除や型変更が伴うため Breaking Change。バージョン運用方針（例: v1 -> v2）を PO と相談し決定する。

## 4. データベース移行方針
- 新マイグレーション（`0004_alter_employees_add_user_id`）を追加。
  - `user_id UUID` カラム追加（`NOT NULL` + `REFERENCES users(id) ON DELETE RESTRICT` を想定）。
  - 一時的に `NOT NULL` 制約を遅延させ、データ移行完了後に制約追加する 2 段階マイグレーションを検討。
  - `UNIQUE (company_id, employee_code)` は維持。必要に応じ `UNIQUE (user_id, company_id)` などの業務ルールを確認。
- バックフィルスクリプト
  - `employees` から `email` が存在するレコードは `users` をメールベースで検索し、存在しなければ `users` に `name`/`status` をセットした新規レコードを挿入。
  - `email` が `NULL` のケースは暫定メール/ID（例: `employee-{id}@placeholder.local`）を発行するか、手動対応リストを K/V で出力する CLI を用意。
  - バックフィル後に `employees.user_id` を更新し、未設定が残ればマイグレーションを失敗させる。
- 移行完了後のマイグレーション（`0005_alter_employees_drop_personal_fields`）
  - `email`, `last_name`, `first_name` カラムと関連チェック制約を削除。
  - `user_id` に `NOT NULL` とインデックス（`idx_employees_user_id`）を付与。
  - ダウングレード手順では元カラム復元＋バックフィル逆変換の難度が高いため、要件に応じて手動手順のドキュメント化で代替する案も提示。

## 5. ドメイン層修正 (`internal/core/employee`)
- エンティティへ `UserID string` を追加し、`Email`/`LastName`/`FirstName` は削除。必要なら `UserSnapshot`（ユーザー情報のコピー）構造体を追加し、表示用途を担保。
- リポジトリインターフェースを `Create(ctx, Employee) (Employee, error)` 等で `UserID` を受け取る形へ変更。
- ユースケースで `internal/core/user` を参照し、`User` の存在ステータス検証（`StatusActive` 以外の扱い）を定義。
- 既存テストを `UserID` 前提に書き換え、社員作成時にユーザー存在チェックをモックで検証する。

## 6. アダプタ層修正
- PostgreSQL リポジトリ (`internal/adapters/repository/postgres/employee_repository.go`)
  - INSERT/UPDATE ステートメントを `user_id` ベースに修正。
  - 個人情報フィールド削除に伴いスキャン対象・構造体マッピングも更新。
  - 必要に応じて `JOIN users` でユーザー情報を引き当て、DTO にまとめる実装パターンを検討。
- gRPC ハンドラ (`internal/adapters/grpc/handler/employee.go`)
  - リクエスト定義を `user_id` 受け取りに変更。従来フィールドは非推奨または削除。
  - レスポンスには少なくとも `user_id` を含め、表示用に `UserSummary`（`users` サービスから取得/組み込み）を返すかを検討。
  - エラーハンドリングを `users` の存在確認失敗時に `NotFound`/`FailedPrecondition` を返すよう整理。
- DI: `internal/platform/server` などで `user.Service` を `employee.Service` へ注入するよう依存関係を更新。

## 7. プロトコル/API 方針
- `proto/employee/v1/employee.proto`
  - `Employee` メッセージから `email`, `last_name`, `first_name` を外し、`string user_id`（必須）と `user.v1.User` をラップする `UserSummary` 追加を検討。
  - `CreateEmployeeRequest` / `UpdateEmployeeRequest` を `user_id` 入力へ統一。Breaking Change 対応として `v2` スキーマの追加も選択肢。
  - 生成コードの更新、`buf breaking` による後方互換チェック、クライアントへのアナウンス手順を作成。
- API ドキュメント (`docs/api/employee-service.md`) を新仕様で更新し、旧フィールドの扱いを明記。

## 8. プラットフォーム & ツール
- `CONFIG_PATH` 配下のシードデータや初期化スクリプト（`assets/seeds`）がある場合、`user_id` へ置き換え。
- CLI/Seeder の追加が必要であれば `cmd/tools/employee_migrator/main.go` などユーティリティを整備し、README に実行手順を追記。
- `docker-compose` が社員データを前提にしている場合は更新し、マイグレーション順序 (`0004` → バックフィル → `0005`) を明示。

## 9. テスト・検証方針
- ユースケース: `internal/core/employee/service_test.go` で `UserService` モックを導入し、`UserID` 検証とエラーパスを網羅。
- リポジトリ: `integration` タグの PG テストを追加し、`user_id` FK 制約と JOIN 挙動を確認。
- gRPC: ハンドラテストで `user_id` 未設定時のバリデーション、非存在ユーザー参照時のエラーマッピングを検証。
- バックフィル: 一時的なデータ移行スクリプトにはユニットテスト or サンプルデータでのスモークテストを付与。
- CI: `buf lint`, `buf generate`, `go test ./...`, `go test -tags=integration ./test/...` の成功を確認し、マイグレーション適用手順を pipeline に反映。

## 10. ドキュメント・ナレッジ
- `README.md` に移行手順、Breaking Change の通知方法、環境変数の更新があれば追記。
- `docs/architecture-overview.md` に社員とユーザーの関係図を加筆。
- 運用手順書（`docs/workflow/` 配下など）へデータ移行チェックリストを追加し、サポートチーム向けに FAQ を整備。

## 11. スケジュールとフォローアップ
- フェーズ1 (0.5日): 要件レビュー、データ移行戦略確定、計画レビュー。
- フェーズ2 (1日): マイグレーション実装・バックフィルスクリプト作成・ローカル検証。
- フェーズ3 (1日): ドメイン/リポジトリ/ハンドラ/プロトコル改修とテスト更新。
- フェーズ4 (0.5日): ドキュメント更新、CI 実行、レビュー対応。
- フォローアップ: `users` ドメインとのイベント連携（ステータス変更通知）、不要となる社員個人情報フィールドの監査ログ削除、今後の SSO / 認証統合の議論を別タスクで起票。
