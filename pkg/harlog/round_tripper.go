package harlog

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"1
)

var _ http.RoundTripper = (*Transport)(nil)

// Transport, HAR formatında HTTP istek/yanıt loglarını toplar.
type Transport struct {
	// Bir sonraki Transport. Eğer nil ise, http.DefaultTransport kullanılır.
	Transport http.RoundTripper
	// Olağandışı (ağ ile ilgili olmayan) bir hata oluştuğunda, bu fonksiyonla hatayı ele alır.
	// Eğer nil ise, log paketi ile hata logu oluşturur ve hatayı görmezden gelir.
	UnusualError func(err error) error

	har   *HARContainer
	mutex sync.Mutex
}

func (h *Transport) init() {
	if h.har != nil {
		return
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.har != nil {
		return
	}

	h.har = &HARContainer{
		Log: &Log{
			Version: "1.2",
			Creator: &Creator{
				Name:    "github.com/vvakame/go-harlog",
				Version: "0.0.1",
			},
		},
	}
}

// HAR, HAR formatında log verilerini döner.
func (h *Transport) HAR() *HARContainer {
	h.init()
	return h.har
}

// RoundTrip, tek bir HTTP işlemi gerçekleştirir ve
// sağlanan İstek için bir Yanıt döner.
func (h *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	h.init()

	baseRoundTripper := h.Transport
	if baseRoundTripper == nil {
		baseRoundTripper = http.DefaultTransport
	}

	entry := &Entry{}
	defer func() {
		h.mutex.Lock()
		h.har.Log.Entries = append(h.har.Log.Entries, entry)
		h.mutex.Unlock()
	}()

	err := h.preRoundTrip(r, entry)
	if err != nil {
		if h.UnusualError != nil {
			err = h.UnusualError(err)
		} else {
			log.Println(err)
			err = nil
		}
		if err != nil {
			return nil, err
		}
	}

	// HTTP yığınına içsel belirli olayların zaman damgalarını kaydetmek için bir izleyici oluştur
	timings, tracer := NewTimingTrace()
	r = r.WithContext(httptrace.WithClientTrace(r.Context(), tracer))

	// HTTP roundtrip işlemini gerçekleştir
	resp, realErr := baseRoundTripper.RoundTrip(r)

	err = nil
	if resp != nil {
		err = h.postRoundTrip(resp, entry)
	}

	timings.endAt = time.Now()
	UpdateEntryWithTimings(entry, timings)

	if err != nil {
		if h.UnusualError != nil {
			err = h.UnusualError(err)
		} else {
			log.Println(err)
			err = nil
		}
		if err != nil {
			return nil, err
		}
	}

	entry.Cache = &Cache{}

	return resp, realErr
}

func (h *Transport) preRoundTrip(r *http.Request, entry *Entry) error {
	var err error
	reqBody := r.Body
	if r.GetBody != nil {
		reqBody, err = r.GetBody()
		if err != nil {
			return err
		}
	}

	var body []byte
	if reqBody != nil {
		body, err = io.ReadAll(reqBody)
		if err != nil {
			return err
		}
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))

	return UpdateEntryWithRequest(entry, r, body)
}

func (h *Transport) postRoundTrip(resp *http.Response, entry *Entry) error {
	respBodyBytes, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))

	UpdateEntryWithResponse(entry, resp, respBodyBytes)
	return nil
}
