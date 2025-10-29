# スキーマ管理行動計画

## 背景と目的
- 既存の SQL マイグレーションファイル運用では全体スキーマの把握が難しく、差分検証も属人的。
- スキーマ定義ファイル（単一ソース）を基点に CLI で差分適用できる体制を整え、変更の可視化・自動化を強化する。

## 成果物と完了条件
- `assets/db/schema.sql`（最新スキーマを表す単一 SQL ファイル）を中心とした宣言的スキーマ管理リポジトリ構成。
- golang-migrate の `schema_migrations` を含む完全なスキーマ定義を保持し、Atlas の差分計算で乖離が出ない状態。
- `atlas.hcl` と `Makefile`（`make atlas-diff`, `make atlas-apply`）を用いたローカル/CI 実行フロー。
- `docs/database.md`（暫定名称）に運用手順とロールバック手順を明記。
- CI で `atlas schema apply --dry-run` が常時実行され、差分が検知されない状態。

## ツール選定
- Atlas（https://atlasgo.io/）を採用。宣言的定義として SQL ファイルを直接扱え、`atlas schema apply` により `schema.sql` と実 DB の差分を自動的に適用可能。
- PostgreSQL/MySQL 両対応であり、必要に応じて `atlas migrate diff` で SQL マイグレーションファイルを生成し、既存資産やレビュー用途にも対応できる。

## フェーズ別計画

### フェーズ 1: 現状調査（2025-10-30 〜 2025-11-05）
- 既存マイグレーションと本番/ステージング DB スキーマを棚卸しし、最新状態を ER 図含めて可視化。
- クリティカルな制約・トリガ・シーケンスなど Atlas で表現が必要な項目を洗い出す。
- Atlas 導入による影響と互換性リスク（ロールバック手段、マイグレーション実行者）を整理し、関係者合意を得る。

### フェーズ 2: PoC 実施とツール整備（2025-11-06 〜 2025-11-12）
- ローカル Docker（`docker/db` 既存設定）上で Atlas CLI をセットアップし、`atlas schema inspect --format sql` で現行 DB から `schema.sql` を生成。
- 生成スキーマをレビューし、必要に応じてコメントやビュー定義などを整理して単一 SQL に統合。
- `Makefile` に Atlas コマンド（`atlas schema apply`, `atlas schema diff`）を組み込み、CI で実行可能にする。
- PoC レポートを `docs/workflow/atlas-poc.md`（暫定）にまとめ、承認を得る。

### フェーズ 3: 本導入と移行（2025-11-13 〜 2025-11-22）
- 既存 SQL マイグレーションと `schema.sql` の整合を確認し、`schema.sql` を唯一のスキーマソースに切り替える。
- PR テンプレートに「`schema.sql` 更新／差分確認」チェックボックスを追加し、レビュー観点を周知。
- CI/CD（GitHub Actions 想定）へ `atlas schema apply --dry-run` と `atlas migrate diff` の自動検証ジョブを追加。
- ステージング環境で `atlas schema apply --auto-approve` を試験実行し、バックアップおよびロールバック手順を確認。

### フェーズ 4: 運用定着とドキュメント整備（2025-11-25 〜 2025-12-03）
- 運用フローを README/開発手順に反映し、チーム向けレクチャーを実施。
- 定期的なスキーマレビュー（例: スプリント毎）を設定し、`schema.sql` のレビュー手順を標準化。
- 本番適用ジョブにアラート/通知（Slack 等）を連携し、失敗時のエスカレーション経路を定義。
- 2025-12-03 時点で Atlas を経由しないスキーマ変更が残っていないことを確認し、完了報告を作成。

## ロールと責任
- Tech Lead: ツール選定承認、CI/CD への組み込みレビュー、運用定着確認。
- Backend 開発者: `schema.sql` の更新、Atlas コマンドを用いたローカル検証、レビュー対応。
- DevOps/Infra: CI/CD 連携、ステージング/本番適用ジョブの設定、ロールバック手順整備。

## リスクと対策
- **リスク**: Atlas 未対応の DB 機能（拡張・カスタムタイプ） → **対策**: PoC 段階で網羅し、必要に応じて SQL マイグレーションとの併用ルールを策定。
- **リスク**: スキーマ定義と実 DB の乖離 → **対策**: CI の `dry-run` と定期的な `atlas schema inspect` を比較し、差異検出を Slack 通知。
- **リスク**: 運用メンバーの習熟不足 → **対策**: 操作手順動画/資料を作成し、初回適用をペア作業で実施。

## チェックリスト
- [ ] `atlas schema inspect --format sql` で生成した `schema.sql` をレビュー済み。
- [ ] `Makefile` に Atlas コマンドが追加され、ローカルで動作確認済み。
- [ ] CI/CD 上で `atlas schema apply --dry-run` が通過することを確認。
- [ ] 運用ドキュメントとロールバック手順が整備され、関係者が把握済み。
- [ ] ステージング/本番で Atlas 経由の適用を完了し、移行報告を提出。
