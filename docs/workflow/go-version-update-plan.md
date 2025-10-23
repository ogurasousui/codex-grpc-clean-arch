# Go バージョン更新計画（1.25.3）

## ゴール
- 開発・CI 環境で利用する Go を最新安定版（1.25.3）へ統一する。
- Go 1.25 系でのビルド・テスト・実行が問題なく行えることを確認し、既存手順書を更新する。
- 将来のバージョンアップを円滑にするため、影響範囲とフォローアップタスクを整理しておく。

## 背景・前提
- アップデート前は `go.mod` で `go 1.24.0` を指定していたが、本計画では Go 1.25 系を標準とする。
- Go 1.25.3 は 2025-10-13 公開の最新安定版であり、セキュリティ修正とコンパイラ最適化が含まれる（参考: https://go.dev/doc/devel/release#go1.25.3）。
- Docker 開発イメージや CI では従来 `golang:1.24-bullseye`、GitHub Actions の `setup-go` では `1.24` を利用していたため、これらも 1.25.3 に揃える。
- Go 1.25 系では `math/rand` の乱数生成改善、`runtime` の GC 調整など一部挙動の変化があるためユースケース単体テストの確認が必要。

## 前提タスク
- 主要ライブラリ（gRPC、protobuf、pgx など）が Go 1.25 をサポートしているかリリースノートを確認する。
- チームメンバーのローカル環境で Go 1.25.3 が利用可能か把握し、必要に応じて asdf / Homebrew / Docker イメージ更新手順を共有する。
- 今回は Go 本体のバージョンアップのみを対象とし、関連ツール（Buf、golangci-lint など）は現状維持とする。

## 作業ステップ
### 1. 調査と準備
- Go 1.25.3 のリリースノートと互換性情報を精読し、破壊的変更がないか確認する。
- 内部 Wiki / Docs で Go 1.24 固有前提が記載されていないか洗い出す。
- Renovate/Dependabot の設定がある場合、Go バージョンの自動更新対象に含まれているか確認する。

### 2. 環境バージョンの更新
- `go.mod` の `go` ディレクティブおよび（必要であれば）`toolchain` ディレクティブを `1.25.3` に更新し、`go mod tidy` を実行する。
- `docker/dev/server/Dockerfile` や他の Dockerfile で利用しているベースイメージを `golang:1.25.3-bookworm`（Debian 12 ベース）に切り替える。
- GitHub Actions（例: `.github/workflows/ci.yml`）の `setup-go` で指定しているバージョンを `1.25.3` に更新する。
- ローカル開発手順（`docs/workflow/local-setup.md`、`README.md` など）に記載されている Go バージョンを更新し、手順の差異がないか検証する。

### 3. 動作確認
- `go test ./...`、`buf lint`、`buf generate`（変更がある場合）を実行し、ビルドとテストが成功することを確認する。
- Docker Compose（`docker compose --profile local up server`）で gRPC サーバーを起動し、ログに警告やエラーが出ていないか確認する。
- Postgres など周辺サービスとの接続テスト、主要ユースケースの疎通確認を実施する。
- 必要に応じて `golangci-lint run` を実行し、 lint エラーが発生しないか確認する。

### 4. ドキュメントとコミュニケーション
- バージョンアップの背景・変更点・検証手順を Pull Request の説明欄に記載し、関連 Issue にリンクする。
- `docs/workflow/branching.md` へバージョンアップ作業時のブランチ命名例やレビュー対象レイヤーを追記する（必要であれば）。
- チーム内告知（Slack / Notion 等）でローカル環境更新手順を共有し、サポートが必要なメンバーをフォローする。

### 5. リリース後フォロー
- バージョンアップ直後のログ監視やエラーレポートを注視し、問題発生時はロールバック方針（Docker タグと go.mod を 1.24 系へ戻す）を明確化する。
- Go 1.26 以降の開発ロードマップを定期的に確認し、次回バージョンアップサイクル（例: 四半期ごと）を決める。
- Renovate/Dependabot の自動化設定が未導入であれば、Go バージョン検知の Issue を起票する。

## 成果物
- `go.mod` / `go.sum`（Go バージョン更新 + 依存関係再整備）
- `docker/dev/server/Dockerfile` およびその他 Dockerfile（ベースイメージ更新）
- `.github/workflows/ci.yml`（Go バージョン更新）
- `README.md` / `docs/workflow/local-setup.md` / `docs/minimal-api-plan.md` などのドキュメント修正
- Pull Request（Conventional Commit 例: `chore: bump go toolchain to 1.25.3`）

## リスクと対応策
- **ビルド時間の増加**: Go 1.25 でコンパイル時間が伸びる可能性 → CI のキャッシュ設定や `GOMAXPROCS` の調整で緩和。
- **依存ライブラリの互換性問題**: `golangci-lint` や gRPC の最新版が Go 1.25 に追従していない場合 → バージョン固定・代替バージョンの検証をチケット化。
- **ローカル環境のアップデート漏れ**: チームメンバーが以前のバージョンを利用し続けるリスク → Makefile に `go env` チェックを追加し、CI で `go version` を明示的に検証する。

## ロールバック戦略
- `git` 上でバージョンアップコミットを revert し、`go.mod` / Dockerfile / CI 設定を 1.24 系へ即時戻す。
- Docker イメージのタグ管理を行い、`golang:1.24-bullseye` を再指定できるようにしておく。
- 問題の再現条件と暫定対応を Issue に記録し、根本原因を調査するタスクを作成する。
