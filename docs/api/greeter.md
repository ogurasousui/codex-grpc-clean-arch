# GreeterService API

## エンドポイント概要
- **サービス名**: `greeter.v1.GreeterService`
- **メソッド**: `SayHello`
- **RPC タイプ**: Unary
- **エンドポイント**: `/greeter.v1.GreeterService/SayHello`

## リクエスト
| フィールド | 型 | 説明 |
|-----------|----|------|
| (なし)    | `google.protobuf.Empty` | 入力値は不要です。 |

## レスポンス
| フィールド | 型 | 説明 |
|-----------|----|------|
| `message` | `string` | 初期実装では常に空文字列を返します。 |

### サンプル

```bash
# gRPCurl を使用した例（TLS 無効のローカル環境）
grpcurl -plaintext localhost:50051 greeter.v1.GreeterService.SayHello
# => {}
```

## 検証・リリース手順
- IDL の lint: `docker run --rm -v $PWD:/workspace -w /workspace bufbuild/buf lint`
- 生成コードの再生成: `docker run --rm -v $PWD:/workspace -w /workspace bufbuild/buf generate`
- 後方互換チェック（将来実装予定）: `docker run --rm -v $PWD:/workspace -w /workspace bufbuild/buf breaking --against buf.build/ogurasousui/codex-grpc-clean-arch` (※リモートモジュール公開後)

## 今後の拡張案
- `message` にユーザー固有メッセージを設定するユースケースを実装。
- 認証が必要になった場合は gRPC インターセプターを追加し `docs/security/` に方針を記載。
