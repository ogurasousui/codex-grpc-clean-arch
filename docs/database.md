# データベース運用ガイド

## 方針概要
- `assets/db/schema.sql` を単一ソースとして管理し、Atlas CLI が実 DB とスキーマファイルの差分を計算します。
- `schema.sql` にはアプリケーションのテーブルに加えてマイグレーション管理用の `schema_migrations` テーブルも含め、実環境と同一状態を保持します。
- Atlas の実行環境やマイグレーションディレクトリは `atlas.hcl` で集中管理します。
- 歴史的な変更履歴は従来どおり `assets/migrations`（golang-migrate 形式）に蓄積され、必要に応じて `atlas migrate diff` で自動生成します。

## 前提
- Atlas CLI をローカルにインストールしてください（例: `brew install ariga/tap/atlas`）。
- ローカル Docker の PostgreSQL を利用する場合は `docker compose up -d postgres` で起動します。
- 既定の接続先は `postgres://app_user:app_password@localhost:15432/app_db?sslmode=disable`（`atlas.hcl` 内 `database_url`）です。環境変数 `DATABASE_URL` や Makefile の変数上書きで変更可能です。

## 代表的なコマンド
| 目的 | コマンド | 補足 |
| --- | --- | --- |
| スキーマ差分の確認 | `make atlas-diff` | DB と `schema.sql` の差分を表示します（内部で `docker://postgres/16/dev` を利用）。|
| スキーマの適用 | `make atlas-apply` | 差分を自動適用します。各変更は Atlas が事前に DDL を提示します。|
| 現行 DB からのスナップショット更新 | `make atlas-inspect` | 実 DB を反映して `schema.sql` を上書きします。コミット前に差分を確認してください。|

- 直接 Atlas を利用する場合の例:
  - 差分確認: `atlas schema diff --from postgres://app_user:app_password@localhost:15432/app_db?sslmode=disable --to file://assets/db/schema.sql --dev-url docker://postgres/16/dev`
  - 適用: `atlas schema apply --url postgres://app_user:app_password@localhost:15432/app_db?sslmode=disable --to file://assets/db/schema.sql --dev-url docker://postgres/16/dev --auto-approve`
  - スナップショット更新: `atlas schema inspect --url postgres://app_user:app_password@localhost:15432/app_db?sslmode=disable --format sql > assets/db/schema.sql`

## マイグレーションファイルの生成
1. `assets/db/schema.sql` を更新し、必要なテーブル定義・制約・インデックスを反映します。
2. 差分を確認: `make atlas-diff`
3. マイグレーションを生成: `atlas migrate diff --env local --to file://assets/db/schema.sql`
4. 生成された `assets/migrations` の SQL をレビューし、テストで検証します。

生成後に `make migrate-up` で反映し、アプリケーションテストを実施してください。マイグレーションが不要な小変更（例: 純粋なドキュメント更新）であっても `schema.sql` と実 DB の差分がないことを確認する運用を徹底します。
CI では `atlas schema diff --format '{{ len .Changes }}'` を利用し、差分が検出された場合はジョブを失敗させます。

## 運用チェックリスト
- [ ] プルリクエストでは `schema.sql` と `assets/migrations` の両方をレビューし、差分が意図どおりか確認したか。
- [ ] `make atlas-diff` の結果をキャプチャし、レビューコメントに添付したか。
- [ ] 本番適用前に `atlas schema apply --env <env> --dry-run` を実行し、DDL を承認したか。
- [ ] ロールバック手順（直前のマイグレーション `down` 実行など）をチームで共有済みか。

## よくあるトラブルと対応
- **接続に失敗する**: `DATABASE_URL` の指定を確認し、Docker の PostgreSQL が起動済みかを確かめてください。
- **`schema.sql` の差分が多すぎる**: 直近で `make atlas-inspect` を実行した開発者と連携し、どの時点の DB を反映したのか確認した上で手作業で差分を調整してください。
- **Atlas が未対応の DDL がある**: `assets/migrations` に手書きで追加し、`schema.sql` へもコメント付きで記載します（例: 拡張機能やカスタム型）。必要であれば PoC のフェーズで代替策を検討します。
