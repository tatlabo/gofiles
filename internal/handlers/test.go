package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

type Data struct {
	StartTime string
	EndTime   string
	Duration  string
	Text      string
}

type DataHtml struct {
	Msg    string
	Data   Data
	Status bool
}

func HandleCtx(w http.ResponseWriter, r *http.Request) {

	delayStr := r.URL.Query().Get("delay")
	delay, err := strconv.Atoi(delayStr)
	if err != nil {
		delay = 1000 // default delay in milliseconds
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// r = r.WithContext(ctx)
	d := make(chan DataHtml, 1)

	go func() {
		defer close(d)
		d <- dataR(delay)
	}()

	select {
	case <-ctx.Done():
		var c DataHtml
		c.Msg = "Request timed out"
		c.Status = false

		tmpl.Render(w, "teplate.html", c)
		return
	case res := <-d:
		tmpl.Render(w, "teplate.html", res)
	}

}

func dataR(t int) DataHtml {

	t1 := time.Now()
	time.Sleep(time.Duration(t) * time.Millisecond)
	t2 := time.Now()

	var data DataHtml
	data.Data = Data{
		StartTime: t1.Format("2006-01-02 15:04:05"),
		EndTime:   t2.Format("2006-01-02 15:04:05"),
		Duration:  t2.Sub(t1).String(),
		Text:      "some data",
	}
	data.Status = true
	data.Msg = "Processing completed successfully"
	return data
}
