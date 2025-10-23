# Air ホットリロード導入計画

## ゴール
- ローカル開発時に `docker compose --profile local up` で起動する API サーバーが Air によってコード変更を自動反映できるようにする。
- Go モジュールキャッシュとビルド成果物を Docker コンテナ内で効率的に再利用し、再ビルドの待ち時間を最小化する。
- チームの運用ドキュメントにホットリロード手順を追加し、既存の `docker compose` ベース運用と整合させる。

## 前提・制約
- ベースイメージは現行の `golang:1.24-bullseye` を継続利用する（Alpine ベースへの乗り換えは今回は行わない）。
- Air の導入はローカル開発プロファイル（`profiles: [local]`）限定とし、本番相当の実行経路には影響を与えない。
- 既存の `CONFIG_PATH=assets/local-compose.yaml` を利用する構成を保ちつつ、サーバー起動コマンドのみ Air 経由に置き換える。

## 作業ステップ
### 1. 開発用 Dockerfile 整備
- `docker/dev/server/Dockerfile`（新規）を作成し、以下を実装する。
  - ベースに `golang:1.24-bullseye` を採用。
  - `curl` / `git` など Air のインストールに必要なパッケージを `apt-get` で追加。
  - `go install github.com/air-verse/air@v1.52.3` など安定版バージョンを固定してインストール。
  - Go ビルドキャッシュ用に `/go/pkg/mod` と `/root/.cache/go-build` を明示的にボリュームマウント可能にしておく（`VOLUME` 宣言 or compose で対応）。

### 2. docker-compose の更新
- `docker-compose.yaml` の `server` サービスに `build` セクションを追加し、上記 Dockerfile を参照する。
- Air を用いた起動コマンドへ変更する（例: `command: air -c .air.toml`）。
- Go モジュールキャッシュとビルドキャッシュ向けの匿名ボリュームを追加し、ホットリロード時のビルド時間を短縮する。
- 既存の `profiles: [local]`・環境変数・ポート設定は維持する。

### 3. Air 設定ファイル導入
- リポジトリルートに `.air.toml`（新規）を追加し、以下を設定する。
  - `cmd = "go build -o tmp/bin/server ./cmd/server"` とし、ビルド成果物を `tmp/bin` に配置。
  - `bin = "tmp/bin/server"` を指定し、`CONFIG_PATH` は環境変数で渡す。
  - `watch_dir` に `cmd`, `internal`, `pkg`, `proto` を指定し、`tmp`, `.git`, `vendor` などは除外。
  - プロファイル切り替え用に `ENV` で `CONFIG_PATH` のデフォルト値を `assets/local-compose.yaml` に設定。

### 4. Makefile / スクリプト整備
- `Makefile` に `dev-up`（Air を利用したローカル起動）、`dev-down`（後片付け）ターゲットを追加。
- 必要に応じて `docker compose --profile local up server` をラップし、Air 導入後も既存コマンド利用者が迷わないようにする。

### 5. ドキュメント更新
- `docs/workflow/local-setup.md` に Air 導入後の起動手順とホットリロードの挙動、推奨コマンドを追記。
- `README.md` の Quick Start に Air を使った開発フローを簡潔に紹介し、初回セットアップ時の手順（例: `docker compose build server`）も明記。
- 必要に応じて `docs/workflow/branching.md` や他の関連ドキュメントへ参照リンクを追加。

### 6. 動作検証
- `docker compose --profile local build server` 後に `docker compose --profile local up server` を実行し、ソースコード更新がホットリロードされることを確認。
- ファイル変更時に Air が再ビルド → サーバー再起動を行い、ログにエラーが出ないことをチェック。
- `docker compose --profile local up` で postgres との連携が維持されているかを確認し、DB 接続エラーなどが発生しないか検証。

## リスクとフォローアップ
- Air のバージョン固定により将来の脆弱性対応が必要になる可能性 → Renovate/Dependabot 対応や手動での定期見直しを検討。
- Docker イメージのビルド時間増加 → キャッシュボリュームの導入と分割ビルドステージ（例: builder + runtime）の検討余地あり。
- 本番イメージとの差異拡大 → 別途プロダクション向け Dockerfile を作成し、`docker compose` で切り替え可能にするフォローアップタスクを記録する。

## 成果物一覧
- `docker/dev/server/Dockerfile`
- `.air.toml`
- `docker-compose.yaml`（修正）
- `Makefile`（修正）
- `docs/workflow/local-setup.md` / `README.md`（修正）
