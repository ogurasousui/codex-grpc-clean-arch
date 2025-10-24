# 会社管理機能実装行動計画

## 1. 背景と目的
- 事業拡大に伴い、顧客・社内サービス双方で会社情報を CRUD できる機能が必要となっている。
- 既存の Clean Architecture 構成（gRPC + PostgreSQL）を踏襲し、新たに `companies` テーブルと gRPC API 群（Create/Get/List/Update/Delete）を整備する。
- ユーザー機能と同等の品質基準（テストカバレッジ、lint、マイグレーション運用）を満たし、将来の連携や拡張に備える。

## 2. 現状整理
- プロジェクトには `users` テーブルおよび関連ユースケースのみが存在し、会社ドメインは未実装。
- gRPC は `proto/user/v1/user.proto` と `proto/greeter/v1/greeter.proto` のみ定義され、会社用のスキーマが無い。
- ドメイン層 (`internal/core`)、リポジトリ層 (`internal/adapters/repository/postgres`)、ハンドラ層 (`internal/adapters/grpc/handler`) もユーザー機能に限定されている。
- マイグレーションは `assets/migrations` に SQL を配置し、`cmd/migrate` で実行する運用が整っている。

## 3. 対象範囲と除外
- 対象: 会社ドメインのエンティティ・ユースケース・リポジトリ・gRPC ハンドラ・プロトコル定義・マイグレーション・テスト。
- 対象外: 既存ユーザードメインとの関連付け、権限管理、外部公開 API Gateway、管理 UI、監査ログ拡張など。
- 将来的な連携が視野にあるため、拡張しやすいスキーマ・API 設計を意識するが、今回のスコープには含めない。

## 4. データベース設計 & マイグレーション
- 新規 `assets/migrations/0002_create_companies.{up,down}.sql` を追加し、以下のカラムを想定。
  - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
  - `name TEXT NOT NULL`
  - `code TEXT NOT NULL UNIQUE`（社内管理用コード、URL slug 等に活用）
  - `status TEXT NOT NULL DEFAULT 'active'`（`active` / `inactive`）
  - `description TEXT`（任意、NULL 可）
  - `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
  - `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`
- `idx_companies_code` など検索に必要なユニークインデックスを作成。
- 既存 seed/fixtures への影響を精査し、必要であれば `assets/seeds/0002_seed_companies` を追加する。
- マイグレーション実施後、`docker compose run --rm migrate up` で適用確認を行う。

## 5. プロトコル定義
- `proto/company/v1/company.proto` を新設し、以下の RPC を定義。
  - `CreateCompany(CreateCompanyRequest) returns (CreateCompanyResponse)`
  - `GetCompany(GetCompanyRequest) returns (GetCompanyResponse)`
  - `ListCompanies(ListCompaniesRequest) returns (ListCompaniesResponse)`（ページング対応）
  - `UpdateCompany(UpdateCompanyRequest) returns (UpdateCompanyResponse)`
  - `DeleteCompany(DeleteCompanyRequest) returns (google.protobuf.Empty)`
- リクエスト/レスポンスは `Company` メッセージを共通利用し、`page_size`/`page_token`/`status_filter` 等を設計。
- `buf lint` でスタイル検証し、`buf generate` or `go generate ./...` で gRPC スタブを生成。`internal/adapters/grpc/gen/company/v1` 配下に生成物を追加。

## 6. ドメイン層 (`internal/core/company`)
- `Company` エンティティ、`Status` 列挙、入力 DTO (`CreateCompanyInput` など) を定義。
- リポジトリインターフェース `Repository` に `Create/Update/Delete/FindByID/FindByCode/List` を用意、ページング結果には `next_page_token` を返却。
- サービス `Service` を実装し、以下のユースケースを提供。
  - `CreateCompany`: `code` の正規化と重複チェック、名称必須、`status` 初期値。
  - `GetCompany`: ID バリデーション、存在しない場合の `ErrCompanyNotFound`。
  - `ListCompanies`: ページサイズ制限、`status` フィルタ、`next_page_token` の算出。
  - `UpdateCompany`: 任意項目更新、`status` バリデーション。
  - `DeleteCompany`: 物理削除（将来のソフトデリートに備え拡張しやすい構造）。
- 80% 以上のカバレッジを目標にユニットテストを作成（フェイクリポジトリ + トランザクションマネージャのスタブ）。

## 7. アダプタ層
- gRPC ハンドラ `internal/adapters/grpc/handler/company.go` を新規追加し、ユースケースを DI してリクエスト/レスポンス変換とエラーマッピングを実装。
  - `InvalidArgument`, `NotFound`, `AlreadyExists` など gRPC ステータスコードに対応。
  - ハンドラの単体テストを `handler/company_test.go` に追加。
- PostgreSQL リポジトリ `internal/adapters/repository/postgres/company_repository.go` を実装。
  - SQL で CRUD とページング (`LIMIT/OFFSET`) を対応。
  - `List` の `status` フィルタ、`next_page_token` 生成ロジックを実装。
  - DB 向け単体テストを `user_repository_test.go` を参考に作成し、`docker` コンテナを利用した実行を想定。
- 生成済み gRPC コードと DI 配線を更新し、コンパイルが通るようにする。

## 8. プラットフォーム & エントリーポイント
- `cmd/server/main.go` で会社ユースケース・リポジトリ・ハンドラを組み立て gRPC サーバーに登録。
- `internal/platform/server/server.go` など DI 設備に会社サービスの初期化処理を追加（`RegisterCompanyService` 仮称）。
- 設定ファイル (`assets/local.yaml`) への追加が不要か確認し、必要に応じて `company` セクションを設ける（現時点では共通 DB 設定を使い回す想定）。

## 9. テスト・検証方針
- ドメイン層: `testing` + `testify` によるユニットテストで正常系・異常系を網羅。
- アダプタ層: gRPC ハンドラはモックユースケースで振る舞い検証、PostgreSQL リポジトリは実 DB テストで CRUD とページングを確認。
- 統合テスト: `test/companies` などを追加し、Docker 上の DB と gRPC クライアントを組み合わせた E2E を `integration` タグで実行。
- CI/ローカル確認手順に `buf lint`, `go test ./...`, `docker run --rm -v $PWD:/app -w /app golang:1.22-bullseye go test ./...` を追記。

## 10. ドキュメント & ナレッジ共有
- `docs/api/company-service.md` を作成し、各 RPC のリクエスト/レスポンス例と gRPCurl サンプルを記載。
- `README.md` に会社機能のテーブル構成・起動手順・API 概要を追記。
- 行動計画承認後、進捗に応じて PR を分割（例: マイグレーション → ドメイン → アダプタ → 統合テスト）し、各 PR の検証手順を明記。

## 11. 想定スケジュールとフォローアップ
- フェーズ 1: スキーマ・プロトコル・ドメイン設計（1～2 日）
- フェーズ 2: リポジトリ実装およびユースケース実装（2 日）
- フェーズ 3: gRPC ハンドラ・統合テスト・ドキュメント整備（1～2 日）
- フェーズ 4: レビュー対応とリリース準備（1 日）
- 未決事項: ステータス値の拡張要件、将来的なユーザー連携（会社-ユーザー関連テーブル）、ソフトデリート導入可否について別途検討する。
