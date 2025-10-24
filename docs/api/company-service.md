# CompanyService API

会社エンティティの作成・取得・一覧・更新・削除を扱う gRPC API です。サービス名は `company.v1.CompanyService` です。

## Proto パス
- ファイル: `proto/company/v1/company.proto`
- go_package: `internal/adapters/grpc/gen/company/v1`

## RPC 一覧

| RPC | リクエスト | レスポンス | 説明 |
| --- | --- | --- | --- |
| `CreateCompany` | `CreateCompanyRequest` | `CreateCompanyResponse` | 会社名とコードを受け取り新規登録します。コード重複時は `ALREADY_EXISTS` を返します。|
| `GetCompany` | `GetCompanyRequest` | `GetCompanyResponse` | `id` で指定された会社を返します。存在しない場合は `NOT_FOUND` を返します。|
| `ListCompanies` | `ListCompaniesRequest` | `ListCompaniesResponse` | ページネーション付きで会社一覧を返します。`page_size` は最大 200 件、`status` でフィルタ可能です。|
| `UpdateCompany` | `UpdateCompanyRequest` | `UpdateCompanyResponse` | `id` をキーに会社情報を更新します。`name`・`code`・`description` は `google.protobuf.StringValue` で指定、`status` は列挙値を利用します。|
| `DeleteCompany` | `DeleteCompanyRequest` | `DeleteCompanyResponse` | `id` で指定された会社を削除します。存在しない場合は `NOT_FOUND` を返します。|

## メッセージ概要

```protobuf
message Company {
  string id = 1;
  string name = 2;
  string code = 3;            // URL slug 等に利用可能なコード（小文字/ハイフン/アンダースコア）
  CompanyStatus status = 4;   // active / inactive
  google.protobuf.StringValue description = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
}

message CreateCompanyRequest {
  string name = 1;                         // 必須
  string code = 2;                         // 必須・ユニーク
  google.protobuf.StringValue description = 3; // 任意（JSON では "description":"..." と指定）
}

message ListCompaniesRequest {
  int32 page_size = 1;   // 0 の場合は既定値 50
  string page_token = 2; // 前回レスポンスの next_page_token を指定
  CompanyStatus status = 3; // フィルタ（未指定=全件）
}

message UpdateCompanyRequest {
  string id = 1;                               // 必須
  google.protobuf.StringValue name = 2;        // 任意更新
  google.protobuf.StringValue code = 3;        // 任意更新（重複不可）
  CompanyStatus status = 4;                    // 任意更新（ACTIVE/INACTIVE）
  google.protobuf.StringValue description = 5; // 任意更新（空文字指定でクリア）
}

message DeleteCompanyResponse {}
```

`CompanyStatus` は次のいずれかを取ります。

- `COMPANY_STATUS_ACTIVE`
- `COMPANY_STATUS_INACTIVE`

`ListCompaniesResponse.next_page_token` は次ページ取得用のオフセット文字列です（未使用時は空文字）。
`CreateCompanyRequest.description` / `UpdateCompanyRequest.description` は JSON では単なる文字列で指定します（例: `"description":"B2B SaaS"`）。空文字を指定すると既存の説明がクリアされます。

## gRPCurl サンプル

### CreateCompany
```bash
grpcurl -plaintext -d '{"name":"Example Inc.","code":"example-inc","description":"B2B SaaS"}' localhost:50051 company.v1.CompanyService/CreateCompany
```

### GetCompany
```bash
grpcurl -plaintext -d '{"id":"<COMPANY_ID>"}' localhost:50051 company.v1.CompanyService/GetCompany
```

### ListCompanies
```bash
grpcurl -plaintext -d '{"page_size":50,"status":"COMPANY_STATUS_ACTIVE"}' localhost:50051 company.v1.CompanyService/ListCompanies
```

### UpdateCompany
```bash
grpcurl -plaintext -d '{"id":"<COMPANY_ID>","code":"example-us","status":"COMPANY_STATUS_INACTIVE","description":""}' localhost:50051 company.v1.CompanyService/UpdateCompany
```

### DeleteCompany
```bash
grpcurl -plaintext -d '{"id":"<COMPANY_ID>"}' localhost:50051 company.v1.CompanyService/DeleteCompany
```

## エラーハンドリング

- バリデーションエラー（名前・コードの空文字、コード形式不正、ページサイズ上限超過、ページトークン不正など）は `INVALID_ARGUMENT`。
- コード重複は `ALREADY_EXISTS`。
- 会社未存在は `NOT_FOUND`。
- それ以外は `INTERNAL` として返却します。
