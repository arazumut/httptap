package harlog

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// UpdateEntryWithTimings, bir HTTP round trip'ten gelen zamanlamalarla bir HAR girişini doldurur
func UpdateEntryWithTimings(entry *Entry, trace *TimingTrace) {
	entry.StartedDateTime = Time(trace.startAt)
	entry.Time = Duration(trace.endAt.Sub(trace.startAt))
	entry.Timings = &Timings{
		Blocked: Duration(trace.startAt.Sub(trace.connStart)),
		DNS:     -1,
		Connect: -1,
		Send:    Duration(trace.writeRequest.Sub(trace.connObtained)),
		Wait:    Duration(trace.firstResponseByte.Sub(trace.writeRequest)),
		Receive: Duration(trace.endAt.Sub(trace.firstResponseByte)),
		SSL:     -1,
	}
	if !trace.dnsStart.IsZero() {
		entry.Timings.DNS = Duration(trace.dnsEnd.Sub(trace.dnsStart))
	}
	if !trace.connStart.IsZero() {
		entry.Timings.Connect = Duration(trace.connObtained.Sub(trace.connStart))
	}
	if !trace.tlsHandshakeStart.IsZero() {
		entry.Timings.SSL = Duration(trace.tlsHandshakeEnd.Sub(trace.tlsHandshakeStart))
	}
}

// UpdateEntryWithRequest, bir HTTP isteğinden gelen değerlerle bir HAR girişini doldurur. Sağlanan
// gövdeyi HTTP isteğinin gövdesi olarak kabul eder ve r.Body'yi okumaz veya değiştirmez.
func UpdateEntryWithRequest(entry *Entry, r *http.Request, body []byte) error {
	bodySize := -1
	var postData *PostData

	if body != nil {
		bodySize = len(body)

		mimeType := r.Header.Get("Content-Type")
		postData = &PostData{
			MimeType: mimeType,
			Params:   []*Param{},
			Text:     string(body),
		}

		// Eksik veya hatalı mime türünü burada yoksay
		mediaType, mediaParams, _ := mime.ParseMediaType(mimeType)

		switch mediaType {
		case "application/x-www-form-urlencoded":
			formdata, err := url.ParseQuery(string(body))
			if err == nil {
				for k, v := range formdata {
					for _, s := range v {
						postData.Params = append(postData.Params, &Param{
							Name:  k,
							Value: s,
						})
					}
				}
			}

		case "multipart/form-data": // "multipart/mixed" türüne de izin vermeyi düşünebilirsiniz
			boundary, ok := mediaParams["boundary"]
			if !ok {
				return fmt.Errorf("boundary içermeyen multipart/form-data isteği alındı")
			}

			mr := multipart.NewReader(bytes.NewReader(body), boundary)
			formdata, err := mr.ReadForm(10 * 1024 * 1024)
			if err == nil {
				for k, v := range formdata.Value {
					for _, s := range v {
						postData.Params = append(postData.Params, &Param{
							Name:  k,
							Value: s,
						})
					}
				}
				for k, v := range formdata.File {
					for _, s := range v {
						postData.Params = append(postData.Params, &Param{
							Name:        k,
							FileName:    s.Filename,
							ContentType: s.Header.Get("Content-Type"),
						})
					}
				}
			}
		}
	}

	entry.Request = &Request{
		Method:      r.Method,
		URL:         r.URL.String(),
		HTTPVersion: r.Proto,
		Cookies:     toHARCookies(r.Cookies()),
		Headers:     toHARNVP(r.Header),
		QueryString: toHARNVP(r.URL.Query()),
		PostData:    postData,
		HeadersSize: -1, // TODO
		BodySize:    bodySize,
	}

	return nil
}

// UpdateEntryWithResponse, bir HTTP yanıtından gelen verilerle bir HAR girişini doldurur. Sağlanan
// gövde baytlarını yanıtın içeriği olarak kabul eder. resp.Body'den okumaz veya değiştirmez.
func UpdateEntryWithResponse(entry *Entry, resp *http.Response, body []byte) {
	mimeType := resp.Header.Get("Content-Type")

	// mime türünü ayrıştır, ve ayrıştırma hatalarını yoksay
	mediaType, _, _ := mime.ParseMediaType(mimeType)

	var text string
	var encoding string
	switch {
	case strings.HasPrefix(mediaType, "text/"):
		text = string(body)
	default:
		text = base64.StdEncoding.EncodeToString(body)
		encoding = "base64"
	}

	entry.Response = &Response{
		Status:      resp.StatusCode,
		StatusText:  "",
		HTTPVersion: resp.Proto,
		Cookies:     toHARCookies(resp.Cookies()),
		Headers:     toHARNVP(resp.Header),
		Content: &Content{
			Size:        resp.ContentLength, // TODO Sıkıştırılmışsa takip et
			Compression: 0,
			MimeType:    mimeType,
			Text:        text,
			Encoding:    encoding,
		},
		RedirectURL: resp.Header.Get("Location"),
		HeadersSize: -1,
		BodySize:    resp.ContentLength,
	}
}

func toHARCookies(cookies []*http.Cookie) []*Cookie {
	harCookies := make([]*Cookie, 0, len(cookies))

	for _, cookie := range cookies {
		harCookies = append(harCookies, &Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Expires:  Time(cookie.Expires),
			HTTPOnly: cookie.HttpOnly,
			Secure:   cookie.Secure,
		})
	}

	return harCookies
}

func toHARNVP(vs map[string][]string) []*NVP {
	nvps := make([]*NVP, 0, len(vs))

	for k, v := range vs {
		for _, s := range v {
			nvps = append(nvps, &NVP{
				Name:  k,
				Value: s,
			})
		}
	}

	return nvps
}
