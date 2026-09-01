package utils

import "context"

// clientIPKey — неэкспортируемый тип ключа контекста: исключает коллизии
// со значениями, которые кладут в контекст другие пакеты.
type clientIPKey struct{}

// WithClientIP возвращает контекст с проставленным IP-адресом клиента.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}

// ClientIP достаёт IP-адрес клиента из контекста.
// Возвращает пустую строку, если мидлвар адрес не проставил, — вызывающий
// код не должен паниковать на запросах вне HTTP-цепочки (например, в тестах).
func ClientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}
