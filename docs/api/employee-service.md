# EmployeeService API

会社 (`companies` テーブル) に紐づく社員情報を CRUD する gRPC API です。サービス名は `employee.v1.EmployeeService` です。

## Proto パス
- ファイル: `proto/employee/v1/employee.proto`
- go_package: `internal/adapters/grpc/gen/employee/v1`

## RPC 一覧

| RPC | リクエスト | レスポンス | 説明 |
| --- | --- | --- | --- |
| `CreateEmployee` | `CreateEmployeeRequest` | `CreateEmployeeResponse` | 会社 ID・社員コード・ユーザー ID を受け取り新規登録します。コード重複時は `ALREADY_EXISTS`、存在しない会社 ID / ユーザー ID の場合は `NOT_FOUND` を返します。|
| `GetEmployee` | `GetEmployeeRequest` | `GetEmployeeResponse` | `id` で指定された社員を返します。存在しない場合は `NOT_FOUND`。|
| `ListEmployees` | `ListEmployeesRequest` | `ListEmployeesResponse` | 必須の `company_id` で社員一覧を取得します。`page_size`（最大 200）、`status` でフィルタ可能です。|
| `UpdateEmployee` | `UpdateEmployeeRequest` | `UpdateEmployeeResponse` | `id` をキーに社員情報を更新します。`employee_code` や `user_id` は `google.protobuf.StringValue` で指定し、空文字を渡すと値をクリアします。|
| `DeleteEmployee` | `DeleteEmployeeRequest` | `DeleteEmployeeResponse` | `id` で指定された社員を削除します。存在しない場合は `NOT_FOUND`。|

## メッセージ概要

```protobuf
message Employee {
  string id = 1;
  string company_id = 2;             // 所属会社の UUID
  string employee_code = 3;          // 会社内で一意な社員コード（小文字/数字/ハイフン/アンダースコア）
  // フィールド 4-6 (email/last_name/first_name) は後方互換のため予約済み
  EmployeeStatus status = 7;         // active / inactive
  google.protobuf.StringValue hired_at = 8;        // YYYY-MM-DD 形式
  google.protobuf.StringValue terminated_at = 9;   // YYYY-MM-DD 形式
  google.protobuf.Timestamp created_at = 10;
  google.protobuf.Timestamp updated_at = 11;
  string user_id = 12;               // users テーブルの ID
  UserSummary user = 13;             // レスポンス用のユーザースナップショット（email/name/status）
}

message CreateEmployeeRequest {
  string company_id = 1;                       // 必須
  string employee_code = 2;                    // 必須・会社内でユニーク
  // フィールド 3-5 (email/last_name/first_name) は後方互換のため予約済み
  EmployeeStatus status = 6;                   // 省略時は active
  google.protobuf.StringValue hired_at = 7;    // 任意・YYYY-MM-DD
  google.protobuf.StringValue terminated_at = 8; // 任意・YYYY-MM-DD（hired_at 以降）
  string user_id = 9;                          // 必須・users.id を参照
}

message ListEmployeesRequest {
  string company_id = 1; // 必須
  int32 page_size = 2;   // 0 の場合は既定値 50
  string page_token = 3; // 前回レスポンスの next_page_token
  EmployeeStatus status = 4; // フィルタ（UNSPECIFIED は無視）
}
```

## grpcurl サンプル

```bash
# 社員作成
grpcurl -d '{
  "company_id":"3f6d...",
  "employee_code":"cs-001",
  "user_id":"9b42..."
}' \
  -plaintext localhost:50051 employee.v1.EmployeeService/CreateEmployee

# 社員一覧
grpcurl -d '{"company_id":"3f6d...","page_size":20}' \
  -plaintext localhost:50051 employee.v1.EmployeeService/ListEmployees
```

## 検証手順

1. `buf lint` と `buf generate` を実行し、proto 定義が正しく生成されることを確認します。
2. `docker run --rm -v $PWD:/app -w /app golang:1.22-bullseye go test ./...` でユニットテストおよびリポジトリテストを実行します。
3. `grpcurl` で登録 → 取得 → 一覧 → 更新 → 削除の一連の呼び出しを行い、ステータスコードとレスポンス内容を確認します。
