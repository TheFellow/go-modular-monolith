package ids

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/cedar-policy/cedar-go"
)

var reader = rand.Reader

func New(entityType cedar.EntityType) (cedar.EntityUID, error) {
	var b [8]byte
	if _, err := reader.Read(b[:]); err != nil {
		return cedar.EntityUID{}, err
	}
	idStr := hex.EncodeToString(b[:])
	return cedar.NewEntityUID(entityType, cedar.String(idStr)), nil
}
