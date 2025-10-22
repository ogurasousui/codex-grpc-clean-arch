# UserService API

ユーザーの作成・更新・削除を扱う gRPC API です。サービス名は `user.v1.UserService` です。

## Proto パス
- ファイル: `proto/user/v1/user.proto`
- go_package: `internal/adapters/grpc/gen/user/v1`

## RPC 一覧

| RPC | リクエスト | レスポンス | 説明 |
| --- | --- | --- | --- |
| `CreateUser` | `CreateUserRequest` | `CreateUserResponse` | メールアドレスと名前を受け取りユーザーを新規作成します。メールアドレス重複時は `ALREADY_EXISTS` を返します。 |
| `UpdateUser` | `UpdateUserRequest` | `UpdateUserResponse` | `id` で指定されたユーザーのプロフィールを更新します。`name` は `google.protobuf.StringValue` で、未指定の場合は変更されません。`status` は `USER_STATUS_*` を指定します。 |
| `DeleteUser` | `DeleteUserRequest` | `google.protobuf.Empty` | `id` で指定されたユーザーを削除します。存在しない場合は `NOT_FOUND` を返します。 |

## メッセージ概要

```protobuf
message User {
  string id = 1;
  string email = 2;
  string name = 3;
  UserStatus status = 4; // active / inactive
  google.protobuf.Timestamp created_at = 5;
  google.protobuf.Timestamp updated_at = 6;
}
```

`UserStatus` は次のいずれかを取ります。

- `USER_STATUS_ACTIVE`
- `USER_STATUS_INACTIVE`

## gRPCurl サンプル

### CreateUser
```bash
grpcurl -plaintext -d '{"email":"user@example.com","name":"User"}' localhost:50051 user.v1.UserService/CreateUser
```

### UpdateUser
```bash
grpcurl -plaintext -d '{"id":"<USER_ID>","name":{"value":"New Name"},"status":"USER_STATUS_INACTIVE"}' localhost:50051 user.v1.UserService/UpdateUser
```

### DeleteUser
```bash
grpcurl -plaintext -d '{"id":"<USER_ID>"}' localhost:50051 user.v1.UserService/DeleteUser
```

## エラーハンドリング

- バリデーションエラー（メール形式、空文字など）は `INVALID_ARGUMENT`。
- メール重複は `ALREADY_EXISTS`。
- ユーザー未存在は `NOT_FOUND`。
- それ以外は `INTERNAL` として返却します。
