package middleware

import (
	"net/http"
	"slices"
	"strings"

	"github.com/ArSteven/devconnect/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	ClaveUsuarioID = "usuario_id"
	ClaveRol       = "rol"
)

// RequiereAuth exige un token de acceso válido en "Authorization: Bearer <token>"
// y deja el ID y el rol del usuario en el contexto para los handlers.
func RequiereAuth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			abortar(c, http.StatusUnauthorized, "NO_AUTENTICADO", "inicia sesión para continuar")
			return
		}
		claims, err := auth.ValidarToken(token)
		if err != nil {
			abortar(c, http.StatusUnauthorized, "NO_AUTENTICADO", "sesión inválida o vencida")
			return
		}
		c.Set(ClaveUsuarioID, claims.Subject)
		c.Set(ClaveRol, claims.Rol)
		c.Next()
	}
}

// RequiereRol deja pasar solo a los roles indicados. Va siempre después de RequiereAuth.
func RequiereRol(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !slices.Contains(roles, c.GetString(ClaveRol)) {
			abortar(c, http.StatusForbidden, "ROL_NO_PERMITIDO", "tu tipo de cuenta no puede hacer esto")
			return
		}
		c.Next()
	}
}

func abortar(c *gin.Context, estado int, codigo, mensaje string) {
	c.AbortWithStatusJSON(estado, gin.H{"error": mensaje, "codigo": codigo})
}
