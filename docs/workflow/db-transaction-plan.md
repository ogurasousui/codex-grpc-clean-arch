# DB トランザクション導入計画

## 背景
- 現状の `internal/adapters/repository/postgres` 実装は `pgxpool.Pool` を直接利用し、クエリ単位で接続を取得している。
- これにより一連のユースケース処理を同一トランザクションで保護できないため、整合性確保や同時実行制御の余地が限定的になっている。
- 今後の拡張（複数テーブルの更新、読み取り一貫性の担保など）に備え、トランザクション境界をアプリケーション層で制御できるようにする。

## 目的
- Read Only トランザクションと Read Write トランザクションを用途に応じて使い分けられる仕組みを導入する。
- ユースケース実行時にトランザクション境界を明示し、リポジトリ層で共通的に利用できるようにする。
- gRPC ハンドラや CLI など、アプリケーションサービスから統一的に呼び出せる API を提供する。

## 要件
- Read Only: `SELECT` 系の処理を一貫性のあるスナップショットで実行し、不要なロック取得を避ける。
- Read Write: 更新や削除を伴う処理を ACID 特性のあるトランザクション内で実行する。
- コンテキストを通じてトランザクションハンドルを共有し、リポジトリは接続プールとトランザクションのどちらからでもクエリできるようにする。
- トランザクション終了後のエラー処理（コミット／ロールバック）をインフラ層で吸収し、ユースケース層ではビジネスロジックに集中できるようにする。

## 現状整理
- `internal/platform/db/postgres/pool.go` で `pgxpool.Pool` を生成し、リポジトリの依存として渡している。
- `UserRepository` は `pgxpool.Pool` と互換なインターフェース（`Query`, `QueryRow`, `Exec`）に依存しており、トランザクション内での実行を考慮していない。
- ユースケース層（`internal/core/user`）はトランザクションの存在を意識しておらず、リポジトリ呼び出しを逐次実行している。

## 基本方針
1. **トランザクションマネージャの導入**
   - `internal/platform/db/postgres` に `TransactionManager` を追加し、`WithinReadOnly(ctx, fn)` / `WithinReadWrite(ctx, fn)` の 2 種類を提供する。
   - `pgx.TxOptions` を利用して `AccessMode` を `pgx.ReadOnly` / `pgx.ReadWrite` に切り替える。
2. **コンテキストへの接続共有**
   - トランザクション開始時に `pgx.Tx` をコンテキストへ埋め込み、リポジトリが取得できるようにする。
   - コネクション未設定時は従来通りプールから直接クエリを実行する。
3. **リポジトリの抽象化**
   - `UserRepository` が依存するインターフェースを `Queryer`（`Query`, `QueryRow`, `Exec`）に拡張し、`pgx.Tx` と `pgxpool.Pool` を両方受け付けられるようにする。
   - トランザクションが存在する場合は `Tx` を利用し、存在しない場合はプールを利用する。
4. **ユースケース／ハンドラからの利用方法を整理**
   - gRPC ハンドラでトランザクションマネージャを呼び出し、コールバックの中でユースケースを実行するスタイルへ移行する。
   - 読み取り系（`GetUser`, `ListUsers` 等）は Read Only、更新系（`CreateUser`, `UpdateUser`, `DeleteUser` 等）は Read Write を選択する。

## 実装ステップ案
1. `TransactionManager` インターフェースと実装の追加。
2. トランザクション中に利用する `Queryer` インターフェースおよびコンテキストキーの定義。
3. `UserRepository` を `Queryer` 対応にリファクタリングし、テストを調整。
4. gRPC ハンドラ／ユースケース呼び出し部をトランザクション管理コードでラップ。
5. トランザクション管理のユニットテストと、主要ユースケースの統合テストを追加。

## 実装サマリ（2025-10-23 更新）
- `internal/platform/db/postgres/transaction.go` に `TransactionManager` と `QueryerFromContext` を追加し、Read Only / Read Write モードを切り替え可能にした。
- `internal/core/user/service.go` が各ユースケースで `WithinReadOnly`/`WithinReadWrite` を利用してトランザクション境界を確立。
- gRPC ハンドラはユースケースのインターフェースを通じて呼び出すだけで、トランザクションモードはユースケース側で選択される。
- `internal/adapters/repository/postgres` では、コンテキストに埋め込まれたトランザクションを優先的に利用するよう調整済み。
- ロールバック挙動を保証する統合テスト（例: `TestUserCRUDIntegration_RollbackOnError`）を追加し、一連の操作がエラー時にコミットされないことを検証。

## テスト方針
- トランザクションマネージャの単体テストで Read Only / Read Write の挙動（コミット・ロールバック）を検証。
- `UserRepository` の既存テストをトランザクションあり／なし双方で動作確認する。
- 統合テストでは gRPC 経由で複数操作を実行し、一括ロールバックが機能するか確認する（必要に応じて `integration` ビルドタグで実行）。

## 懸念・検討事項
- コンテキスト経由での `pgx.Tx` 受け渡しは並行処理での誤用を避けるためスコープ管理を明確化する必要がある。
- 将来的に複数 DB を扱う場合、トランザクションマネージャの抽象化をより一般化する必要がある。
- パフォーマンス面で Read Only トランザクションが不要なケースもあるため、API で選択できる柔軟性を持たせる。
