# 社員管理機能実装行動計画

## 1. 背景と目的
- 会社情報 (`companies` テーブル) の整備が完了したため、会社に紐づく社員情報を管理する仕組みを追加し、組織単位でのリソース運用を可能にする。
- 最低限の CRUD API を gRPC 経由で提供し、既存の Clean Architecture 構成（core/adapters/platform）を踏襲する。
- 将来的な人事系機能（部署管理、権限連携など）へ拡張できるよう、テーブル・API ともに拡張性を確保する。

## 2. 現状整理
- データベースには `users`, `companies` のみが存在し、社員ドメインは未実装。
- `internal/core/company` のサービスと PostgreSQL リポジトリが稼働しており、Company ID をキーにした参照/検査ロジックが整っている。
- gRPC では `proto/company/v1` と `proto/user/v1` が用意されているが、社員用のスキーマが存在しない。
- DI や設定読み込みは `internal/platform` 配下で統一管理されており、新ドメイン追加時は同様の流れに従う必要がある。

## 3. スコープと前提
- 対象: 社員テーブルおよび関連マイグレーション、ドメイン層（エンティティ・ユースケース）、PostgreSQL リポジトリ、gRPC ハンドラと proto 定義、テスト、ドキュメント。
- 除外: 勤怠・給与などの業務ロジック、外部サービス連携、認証・認可拡張、会社との 1:n 以外のリレーション（部署・役職テーブル等）。
- 社員は必ず既存 `companies.id` に所属する前提とし、外部キー制約とアプリケーションレイヤーのバリデーションで担保する。

## 4. データベース設計 & マイグレーション
- `assets/migrations/0003_create_employees.{up,down}.sql` を追加し、以下のカラムを定義。
  - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
  - `company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE`
  - `employee_code TEXT NOT NULL`
  - `email TEXT`
  - `last_name TEXT NOT NULL`
  - `first_name TEXT NOT NULL`
  - `status TEXT NOT NULL DEFAULT 'active'`（`active` / `inactive` を想定）
  - `hired_at DATE`、`terminated_at DATE`
  - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
  - `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- 制約/インデックス
  - `UNIQUE (company_id, employee_code)` で社内コードの一意性を担保。
  - `idx_employees_company_id_status` で会社＋状態フィルタを高速化。
  - メールアドレスについては将来の要件確認まで UNIQUE 制約は設けない。
- ダウングレードではテーブルと関連インデックスを削除し、副作用が残らないようにする。
- 可能であれば `assets/seeds` にサンプルデータを追加し、統合テストで再利用する。

## 5. プロトコル定義
- `proto/employee/v1/employee.proto` を新設。
  - サービス名: `EmployeeService`。
  - RPC: `CreateEmployee`, `GetEmployee`, `ListEmployees`, `UpdateEmployee`, `DeleteEmployee`。
  - `ListEmployees` は `company_id`, `status`, `page_size`, `page_token` を受け取り、`next_page_token` を返却。
  - `Employee` メッセージに社員基本情報と所属会社 ID を含める。
- `buf.yaml` / `buf.gen.yaml` を更新し、生成コードを `internal/adapters/grpc/gen/employee/v1` に出力。
- `buf lint` でスタイルを検証し、`buf breaking` の観点で会社スキーマとの依存が無いことを確認。

## 6. ドメイン層 (`internal/core/employee`)
- 新パッケージを作成し、以下を実装。
  - エンティティ `Employee`、値オブジェクト（`ID`, `CompanyID`, `Status` 等）のバリデーション。
  - `Repository` インターフェース: `Create`, `Update`, `Delete`, `FindByID`, `List`（ページング対応）。
  - ユースケース `Service`（`UseCase` インターフェース）で上記 RPC に対応するメソッドを提供。
- ビジネスルール
  - 会社存在チェック: `company.Repository` 経由またはリポジトリ内の FK 制約エラーを解釈。
  - `employee_code` のフォーマット（英数字 + `_`/`-` のみ）と重複チェック。
  - 退職日 (`terminated_at`) は入社日より後であることを検証。
- テスト
  - フェイクリポジトリを利用したユースケースユニットテストを追加し、カバレッジ 80% を維持。
  - 異常系（存在しない会社 ID、重複コード、無効なステータス）を網羅する。

## 7. アダプタ層
- PostgreSQL リポジトリ: `internal/adapters/repository/postgres/employee_repository.go` を新規追加。
  - `pgx` を用いて CRUD／ページング（`LIMIT` + `OFFSET`）を実装。
  - FK 違反はドメインエラー `ErrCompanyNotFound` として扱う。
  - `List` では `company_id` フィルタ必須、`status` は任意で WHERE 句に追加。
- gRPC ハンドラ: `internal/adapters/grpc/handler/employee.go`。
  - リクエスト/レスポンス変換、バリデーション、エラーマッピング（`InvalidArgument`, `NotFound`, `AlreadyExists` など）。
  - 単体テストでユースケースモックを利用し、入出力およびエラー挙動を確認。
- 既存 DI: `internal/platform/server` などで EmployeeService の登録を追加し、`cmd/server/main.go` から gRPC サーバーへ登録。

## 8. プラットフォーム & 設定
- 既存の DB 接続設定 (`internal/platform/db/postgres`) を再利用し、社員リポジトリへ依存性注入する。
- 必要に応じて `internal/platform/transaction` の共通実装を導入し、会社サービスと同様に `TransactionManager` を利用できるよう整理。
- `CONFIG_PATH` で参照する YAML に追加設定が不要か確認し、ログやメトリクスで社員ドメインの識別子を付与する。

## 9. テスト・検証方針
- ドメイン: フェイクリポジトリ + `testify` を用いて正常系/異常系を網羅。
- リポジトリ: Docker 上の PostgreSQL を利用した実 DB テストを `integration` タグで実施（CRUD + ページング + FK 制約）。
- gRPC: ハンドラ単体テストに加え、`test/employees` に統合テストを追加し、社員作成→取得→一覧→更新→削除の一連を検証。
- CI/ローカル手順:
  - `buf lint`, `buf generate`
  - `docker run --rm -v $PWD:/app -w /app golang:1.22-bullseye go test ./...`
  - `go test -tags=integration ./test/employees`

## 10. ドキュメント & ナレッジ共有
- `docs/api/employee-service.md` を追加し、RPC ごとの I/O 例、`grpcurl` コマンド、利用するステータス値を記載。
- `README.md` に社員機能の概要と動作確認手順を追記し、マイグレーション実行順序 (`0003` 適用) を明記。
- `docs/architecture-overview.md` に社員ドメインのコンテキスト図と依存関係を追加。
- 追加ツールやコマンド（例: `buf lint --path proto/employee/v1`）があれば `AGENTS.md` への反映を検討。

## 11. スケジュールとフォローアップ
- フェーズ 1 (0.5 日): 要件再確認、データモデル・proto 設計、行動計画レビュー。
- フェーズ 2 (1 日): マイグレーション作成、ドメインサービス・リポジトリ実装、ユニットテスト整備。
- フェーズ 3 (1 日): gRPC ハンドラ・統合テスト・ドキュメント更新。
- フェーズ 4 (0.5 日): リファクタ、lint/format、レビュー対応、マージ準備。
- フォローアップ: 部署・役職などの関連マスタや、社員とユーザーの紐付け要件を別途検討し、二重登録防止や認証連携の仕様を詰める。
