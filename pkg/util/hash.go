package util

import (
	"fmt"

	"github.com/mitchellh/hashstructure"
)

func Hash(val interface{}) string {

	hash, _ := hashstructure.Hash(val, nil)
	return fmt.Sprintf("%d", hash)
}
