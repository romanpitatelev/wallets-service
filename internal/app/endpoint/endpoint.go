package endpoint

import (
	"fmt"
	"net/http"
	"time"
)

type Service interface {
	TimeNow() time.Time
}

type Endpoint struct {
	s Service
}

func New(s Service) *Endpoint {
	return &Endpoint{
		s: s,
	}
}

func (e *Endpoint) Status(w http.ResponseWriter, r *http.Request) {
	d := e.s.TimeNow()
	answer := fmt.Sprint("Current date and time:", d.Format("01:02:2006 15:04:05"))
	fmt.Println("Request status:", http.StatusOK)
	fmt.Println(answer)
}
