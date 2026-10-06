package agent

import (
	"errors"
	"fmt"
	"net"
	"net/url"
)

// OutboundIP возвращает локальный IP-адрес, с которого хост скорее всего пойдёт
// на сервер. Используется как значение заголовка X-Real-IP в запросах агента.
// serverAddr — URL сервера со схемой (https://...).
func OutboundIP(serverAddr string) (net.IP, error) {
	u, err := url.Parse(serverAddr)
	if err != nil {
		return nil, fmt.Errorf("error parsing server address: %w", err)
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("no host in server address %q", serverAddr)
	}
	port := u.Port()
	if port == "" {
		port = u.Scheme // у схем есть дефолтный порт
	}
	return OutboundIPHostPort(net.JoinHostPort(u.Hostname(), port))

}

// OutboundIPHostPort принимает host:port без схемы и непосредственно резолвит.
func OutboundIPHostPort(hostport string) (net.IP, error) {
	// поскольку это UDP, реального соединения тут нет
	conn, err := net.Dial("udp", hostport)
	if err != nil {
		return nil, fmt.Errorf("error getting outbound connection: %w", err)
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, errors.New("error while trying to get agent outbound address")
	}

	return localAddr.IP, nil
}
