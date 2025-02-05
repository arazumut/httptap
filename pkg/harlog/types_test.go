package harlog

import (
	"reflect"
	"testing"
	"time"
)

func TestTime_MarshalJSON(t *testing.T) {
	// Zaman dilimini yükle
	tz, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}

	// Test senaryoları
	tests := []struct {
		isim           string
		zaman          Time
		beklenen       string
		hataBekleniyor bool
	}{
		{
			isim:           "normal",
			zaman:          Time(time.Date(2019, 10, 2, 12, 16, 30, 50, tz)),
			beklenen:       `"2019-10-02T12:16:30+09:00"`,
			hataBekleniyor: false,
		},
		{
			isim:           "sıfır değer",
			zaman:          Time(time.Time{}),
			beklenen:       `null`,
			hataBekleniyor: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.isim, func(t *testing.T) {
			sonuc, err := tt.zaman.MarshalJSON()
			if (err != nil) != tt.hataBekleniyor {
				t.Errorf("MarshalJSON() hata = %v, hataBekleniyor %v", err, tt.hataBekleniyor)
				return
			}
			if !reflect.DeepEqual(string(sonuc), tt.beklenen) {
				t.Errorf("MarshalJSON() sonuc = %v, beklenen %v", string(sonuc), tt.beklenen)
			}
		})
	}
}

func TestTime_UnmarshalJSON(t *testing.T) {
	// Zaman dilimini yükle
	tz, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}

	type argümanlar struct {
		veri string
	}
	tests := []struct {
		isim           string
		argüman        argümanlar
		beklenen       Time
		hataBekleniyor bool
	}{
		{
			isim: "normal",
			argüman: argümanlar{
				veri: `"2019-10-02T12:16:31+09:00"`,
			},
			beklenen:       Time(time.Date(2019, 10, 2, 12, 16, 31, 0, tz)),
			hataBekleniyor: false,
		},
		{
			isim: "null",
			argüman: argümanlar{
				veri: `null`,
			},
			beklenen:       Time(time.Time{}),
			hataBekleniyor: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.isim, func(t *testing.T) {
			var v Time
			if err := v.UnmarshalJSON([]byte(tt.argüman.veri)); (err != nil) != tt.hataBekleniyor {
				t.Errorf("UnmarshalJSON() hata = %v, hataBekleniyor %v", err, tt.hataBekleniyor)
			}
			if !time.Time(v).Equal(time.Time(tt.beklenen)) {
				t.Errorf("UnmarshalJSON() sonuc = %v, beklenen %v", v, tt.beklenen)
			}
		})
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		isim           string
		süre           Duration
		beklenen       string
		hataBekleniyor bool
	}{
		{
			isim:           "normal",
			süre:           Duration(10 * time.Millisecond),
			beklenen:       "10",
			hataBekleniyor: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.isim, func(t *testing.T) {
			sonuc, err := tt.süre.MarshalJSON()
			if (err != nil) != tt.hataBekleniyor {
				t.Errorf("MarshalJSON() hata = %v, hataBekleniyor %v", err, tt.hataBekleniyor)
				return
			}
			if !reflect.DeepEqual(string(sonuc), tt.beklenen) {
				t.Errorf("MarshalJSON() sonuc = %v, beklenen %v", string(sonuc), tt.beklenen)
			}
		})
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	type argümanlar struct {
		veri string
	}
	tests := []struct {
		isim           string
		argüman        argümanlar
		beklenen       Duration
		hataBekleniyor bool
	}{
		{
			isim: "normal",
			argüman: argümanlar{
				veri: "10",
			},
			beklenen:       Duration(10 * time.Millisecond),
			hataBekleniyor: false,
		},
		{
			isim: "null",
			argüman: argümanlar{
				veri: `null`,
			},
			beklenen:       Duration(0),
			hataBekleniyor: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.isim, func(t *testing.T) {
			var v Duration
			if err := v.UnmarshalJSON([]byte(tt.argüman.veri)); (err != nil) != tt.hataBekleniyor {
				t.Errorf("UnmarshalJSON() hata = %v, hataBekleniyor %v", err, tt.hataBekleniyor)
			}
			if v != tt.beklenen {
				t.Errorf("UnmarshalJSON() sonuc = %v, beklenen %v", v, tt.beklenen)
			}
		})
	}
}
