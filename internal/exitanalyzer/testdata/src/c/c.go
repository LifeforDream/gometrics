package c

import l "log"

func aliasedLog() {
	l.Fatal("boom") // want "found l.Fatal usage outside main.main"
}

type sample struct {
}

func (s *sample) Fatal() {}

func logVar() {
	log := &sample{}
	log.Fatal()
}
