# Feature Branch & PR 手順

1. `main` ブランチを最新化します。`git checkout main && git pull`。
2. 機能ごとに feature ブランチを作成します。例: `git checkout -b feat/simple-greeter`。
3. 実装・テスト・ドキュメント更新をコミットします。コミットメッセージは Conventional Commits を採用します。
4. `docker compose run --rm server go test ./...` などでテストを実行し、結果を記録します。ホストに Go がなくてもコンテナで完結させる想定です。
5. ブランチをプッシュします: `git push origin feat/simple-greeter`。
6. GitHub で Pull Request を作成し、以下を記載します。
   - 目的と背景（Issue へのリンク）
   - 実装概要とテスト手順
   - 影響範囲とロールバックプラン
   - Buf 生成物を更新した場合は差分ファイルを含める
7. レビュー指摘が解消したら `main` へマージし、ブランチを削除します。
