package helper

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

// cursorPayload adalah posisi terakhir yang dilihat client: kombinasi
// created_at dan id. Keduanya diperlukan (bukan created_at saja) karena
// dua baris bisa punya created_at yang sama persis (dibuat di detik yang
// sama) -- id sebagai tie-breaker menjamin urutan tetap stabil.
type cursorPayload struct {
	CreatedAt time.Time `json:"created_at"`
	ID        int       `json:"id"`
}

// EncodeCursor mengubah posisi terakhir menjadi string buram (base64).
// Client memperlakukannya sebagai token acak -- ia tidak perlu tahu, dan
// tidak seharusnya bergantung pada, apa isi sesungguhnya di dalamnya.
func EncodeCursor(createdAt time.Time, id int) string {
	payload := cursorPayload{CreatedAt: createdAt, ID: id}
	raw, _ := json.Marshal(payload) // payload sederhana, Marshal tidak pernah gagal di sini
	return base64.URLEncoding.EncodeToString(raw)
}

// DecodeCursor membalikkan EncodeCursor. Mengembalikan error bila cursor
// tidak bisa dibaca (rusak, diutak-atik, atau dari versi format lama) --
// fail closed, bukan menebak-nebak posisi mana yang dimaksud.
func DecodeCursor(cursor string) (time.Time, int, error) {
	if cursor == "" {
		return time.Time{}, 0, nil
	}

	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, 0, errors.New("cursor tidak valid")
	}

	var payload cursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return time.Time{}, 0, errors.New("cursor tidak valid")
	}

	return payload.CreatedAt, payload.ID, nil
}
