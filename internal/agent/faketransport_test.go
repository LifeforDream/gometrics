package agent

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
)

// fakeTransport — реализация http.RoundTripper поверх net.Pipe вместо
// настоящих TCP-сокетов. В отличие от httptest.Server, весь обмен идёт по
// каналам net.Pipe, которые создаются внутри вызывающей горутины — если она
// запущена внутри пузыря testing/synctest, эти каналы наследуют пузырь и
// остаются "durably blocked"-совместимыми, так что фейковые часы пузыря
// продолжают продвигаться, даже когда агент реально отправляет HTTP-запрос и
// дожидается ответа.
//
// Хендлер выполняется через httptest.NewRecorder(), то есть без какой-либо
// реальной сети, поэтому уже написанные для httptest.Server http.HandlerFunc
// подходят без изменений.
type fakeTransport struct {
	handler http.Handler
}

func newFakeTransport(handler http.HandlerFunc) *fakeTransport {
	return &fakeTransport{handler: handler}
}

// RoundTrip реализует http.RoundTripper.
func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	srvConn, cliConn := net.Pipe()
	go f.serve(srvConn)

	if err := req.Write(cliConn); err != nil {
		_ = cliConn.Close()
		return nil, err
	}

	return http.ReadResponse(bufio.NewReader(cliConn), req)
}

func (f *fakeTransport) serve(conn net.Conn) {
	defer conn.Close()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		return
	}

	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)

	_ = rec.Result().Write(conn)
}

func newFakeClient(handler http.HandlerFunc) *http.Client {
	return &http.Client{
		Transport: newFakeTransport(handler),
	}
}
