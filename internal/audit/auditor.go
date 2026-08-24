package audit

import (
	"sync"
	"time"

	"go.uber.org/zap"

	models "github.com/LifeforDream/gometrics/internal/model"
)

type Sender interface {
	getID() string
	setChan(chan Event)
	worker()
}

// inputBufferSize ограничивает число событий аудита, ожидающих рассылки.
// Под устойчивой нагрузкой, когда подписчики не успевают вычитывать канал
// (например, зависший HTTPAuditSender), Update отбрасывает новые события
// вместо блокировки вызывающего запроса или неограниченного роста горутин.
const inputBufferSize = 256

// Auditor — субъект (Subject) в паттерне «Наблюдатель». Единственная
// горутина dispatch владеет всеми каналами подписчиков: она и пишет в них, и
// закрывает их
type Auditor struct {
	mu             sync.Mutex
	subChan        map[string]chan Event
	input          chan Event
	dispatcherDone chan struct{}
	logger         *zap.Logger
}

func NewAuditor(logger *zap.Logger) *Auditor {
	a := &Auditor{logger: logger}
	a.input = make(chan Event, inputBufferSize)
	a.dispatcherDone = make(chan struct{})
	go a.dispatch()
	return a
}

func (a *Auditor) RegisterSub(sub Sender) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.subChan == nil {
		a.subChan = make(map[string]chan Event)
	}
	c := make(chan Event)
	// не больше одного sender'а на тип
	_, exists := a.subChan[sub.getID()]
	if exists {
		return
	}
	a.subChan[sub.getID()] = c
	sub.setChan(c)

	go sub.worker()
}

// dispatch читает события из input и рассылает их по снапшоту subChan.
// Когда Close закрывает input, цикл завершается, и dispatch сам закрывает
// каждый канал подписчика — единственный писатель одновременно является
// единственным, кто решает, когда закрывать.
func (a *Auditor) dispatch() {
	defer close(a.dispatcherDone)
	for e := range a.input {
		for _, c := range a.subsSnapshot() {
			c <- e
		}
	}
	for _, c := range a.subsSnapshot() {
		close(c)
	}
}

func (a *Auditor) subsSnapshot() []chan Event {
	a.mu.Lock()
	defer a.mu.Unlock()
	chans := make([]chan Event, 0, len(a.subChan))
	for _, c := range a.subChan {
		chans = append(chans, c)
	}
	return chans
}

func (a *Auditor) Update(metrics []models.Metrics, ipAddress string) {
	a.mu.Lock()
	empty := len(a.subChan) == 0
	in := a.input
	a.mu.Unlock()
	if empty {
		return
	}

	var ae Event
	ae.Ts = time.Now().Unix()
	ae.Metrics = make([]string, 0, len(metrics))
	for _, met := range metrics {
		ae.Metrics = append(ae.Metrics, met.ID)
	}
	ae.IPAddress = ipAddress

	// Неблокирующая отправка в буферизованный input: если буфер заполнен
	// (подписчики не успевают вычитывать), событие отбрасывается вместо
	// блокировки вызывающего запроса или неограниченного роста горутин.
	select {
	case in <- ae:
	default:
		if a.logger != nil {
			a.logger.Warn("audit event dropped: input buffer full", zap.Int("metrics_count", len(ae.Metrics)))
		}
	}
}

func (a *Auditor) Close() {
	close(a.input)
	<-a.dispatcherDone
}
