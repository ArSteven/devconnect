package service

import "strings"

var quitarTildes = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n")

// normalizarTexto deja el texto como lo compara la base: minúsculas, sin tildes y sin
// espacios repetidos. Debe coincidir con repository.sinTildes.
func normalizarTexto(s string) string {
	return quitarTildes.Replace(strings.ToLower(strings.Join(strings.Fields(s), " ")))
}
