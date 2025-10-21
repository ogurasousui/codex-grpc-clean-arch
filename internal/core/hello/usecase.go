package hello

import "context"

// Greeter は挨拶文を生成するユースケースのインターフェースを定義します。
type Greeter interface {
	// SayHello は呼び出し元へ返却するメッセージを生成します。
	// 初期実装では空文字列を返却します。
	SayHello(ctx context.Context) (string, error)
}

// Service は Greeter ユースケースのデフォルト実装です。
type Service struct{}

// NewService は Greeter ユースケースの新しいインスタンスを返します。
func NewService() *Service {
	return &Service{}
}

// SayHello は常に空文字列を返却します。将来的にロジックを追加する際はここを拡張します。
func (s *Service) SayHello(ctx context.Context) (string, error) {
	return "", nil
}
