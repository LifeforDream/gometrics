package audit

// Event — одно событие аудита, публикуемое Auditor.Update и рассылаемое
// подписчикам (Sender).
type Event struct {
	Ts        int64    `json:"ts"`         // unix timestamp
	Metrics   []string `json:"metrics"`    // ["Alloc", "Frees", ...], // наименование полученных метрик
	IPAddress string   `json:"ip_address"` // IP адрес входящего запроса
}
