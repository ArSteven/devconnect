package repository

import (
	"fmt"
	"strings"
)

// sinTildes devuelve la expresión SQL que pasa una columna a minúsculas y sin tildes,
// para comparar "Girón" con "giron" sin instalar extensiones en la base.
// El argumento siempre es un nombre de columna escrito en el código, nunca un dato del usuario.
func sinTildes(columna string) string {
	return fmt.Sprintf("translate(lower(%s), 'áéíóúüñÁÉÍÓÚÜÑ', 'aeiouunaeiouun')", columna)
}

// escaparLike neutraliza los comodines de LIKE para que el texto se busque tal cual.
func escaparLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
