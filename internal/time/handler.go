package time

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type TimeHandler struct{}

func NewTimeHandler(router *chi.Mux) {
	handler := &TimeHandler{}
	router.Get("/time", handler.TimeNow())
}

func (handler *TimeHandler) TimeNow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeCurr := time.Now()
		answer := fmt.Sprint("Current date and time:", timeCurr.Format("01:02:2006 15:04:05"))
		fmt.Println("Request status:", http.StatusOK)
		fmt.Println(answer)
	}
}
