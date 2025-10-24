# GitHub Actions CI 改修行動計画

## 背景
- 現行の `.github/workflows/ci.yml` は `push` と `pull_request` の両方をトリガーとしており、レビュー前の PR に対しても全テストが走っている。
- ワークフロー内で PostgreSQL サービスを直接起動しているが、ローカルと同様の Docker ベース統合テスト手順を再利用できていない。
- 統合テストは `test/` 配下の `integration` ビルドタグ付きテストで Postgres への接続を必要とし、`assets/ci.yaml` の設定値に依存する。

## 目的
- GitHub Actions 上では `pull_request` イベントのみでテストを実行し、対象ブランチへの直接 push ではワークフローを起動しない。
- 統合テストを Docker で起動した Postgres に対して実行し、ローカルの `docker compose` 手順と整合させる。
- 既存のユニットテスト (`go test ./...`) も同じパイプラインで継続的に実行する。

## 対象スコープ
- `.github/workflows/ci.yml` のトリガーおよびステップ構成変更。
- 必要に応じた `docker-compose.yaml` のプロファイル・サービス調整、CI 用設定ファイルの更新。
- 追加スクリプト（例: `scripts/ci/run_integration.sh`）が必要な場合は同リポジトリ内に配置する。
- Buf やその他リンタの導入・変更は今回のスコープ外とする。

## 対応方針
### 1. トリガー整理
- `on.pull_request` のみを残し、`push` トリガーを削除する。
- `pull_request` の対象ブランチ（例: `main`）を明示し、不要なブランチには反応しないようにする。
- 手動実行 (`workflow_dispatch`) やスケジュール実行が必要なら別途検討する。

### 2. ジョブ分割とキャッシュ
- `unit-test` と `integration-test` の 2 ジョブに分け、統合テストはユニットテスト完了後に実行する依存関係 (`needs`) を設定する。
- `actions/setup-go` とモジュールキャッシュ (`actions/cache` or setup-go の内蔵キャッシュ) を共通化し、再ダウンロードを最小限にする。

### 3. Docker ベース統合テスト実行
- ジョブ内で `docker compose -f docker-compose.yaml --profile ci up -d postgres` のように PostgreSQL を起動する。
  - `docker-compose.yaml` に `ci` プロファイルが無い場合は追加するか、既存 `local` プロファイルを流用する。
- DB の起動完了を `docker compose ps` またはヘルスチェック待機スクリプトで確認する。
- 統合テスト実行時は `CONFIG_PATH=$(pwd)/assets/ci.yaml go test -tags integration ./test/...` を実行する。
- テスト終了後 `docker compose down` を実行し、リソースを解放する。

### 4. マイグレーション処理
- 統合テスト前に `go run ./cmd/migrate -config ${CONFIG_PATH} up` を実行し、DB スキーマを最新状態にする。
- シード投入が必要な場合はテストコードに任せる（既存の `applySeeds` が Up を実行するため追加コマンドは不要）。

### 5. 設定ファイル調整
- `assets/ci.yaml` の `database.host` は GitHub Actions 上で `docker compose` からアクセスできるよう `localhost` を維持し、ポートは `docker-compose.yaml` のマッピング (`15432`) と一致させる。
- CI 用に新たな `.env` が必要な場合は `assets/` 配下に追記し、Secrets は GitHub Actions の `env` で管理する。

## 実装手順
1. 新ブランチ `feat/ci-docker-integration` を作成する。
2. `.github/workflows/ci.yml` でトリガーとジョブ構成を更新する。
3. 必要なら `docker-compose.yaml` に `ci` プロファイルを追加し、Postgres サービスのポートやボリュームを CI 用に最適化する。
4. 統合テスト用スクリプトを `scripts/ci/` に追加し、ワークフローステップから呼び出す（再利用性向上のため）。
5. 変更後ローカルで `docker compose --profile local up -d postgres` ⇒ `CONFIG_PATH=assets/local.yaml go test -tags integration ./test` を試行し、動作確認する。
6. GitHub Actions の `workflow_dispatch` やブランチ保護設定との整合性を確認し、必要に応じてドキュメント (`docs/workflow/branching.md` 等) を更新する。

## 検証
- GitHub Actions のテスト実行ログで以下を確認する。
  - `integration-test` ジョブが `unit-test` の成功後に起動する。
  - Docker Compose による Postgres 起動ログが出力され、ヘルスチェック成功後にテストが開始されている。
  - マイグレーションと統合テストが成功し、`go test -tags integration` が 0 exit code を返す。
- 失敗時に備え、`docker compose logs postgres` を出力して原因分析できるようにする。

## リスクと対応策
- **Docker デーモンが利用不可**: GitHub ホステッドランナーで Docker が無効化されているケースは想定しないが、将来的にセルフホストへ移行する場合は対応が必要。
- **テスト時間の増加**: コンテナ起動とマイグレーションで所要時間が伸びる可能性があるため、キャッシュ活用と冪等な `down` 処理で時間短縮を図る。
- **環境変数の漏れ**: 機密情報は使用せず、固定値は `assets/ci.yaml` で管理する。

## 完了条件
- GitHub Actions ワークフローが `pull_request` 時のみ実行される。
- 統合テストが Docker コンテナ上の Postgres を利用して成功する。
- 新しいフローと利用手順を README あるいは関連ドキュメントに追記する（必要な場合）。
