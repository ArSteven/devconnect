package model

// FiltroMetricas son los parámetros de GET /admin/metricas. Sin desde, cuenta toda la historia;
// sin hasta, llega hasta hoy. Los días son de Colombia.
type FiltroMetricas struct {
	Desde string `form:"desde" binding:"omitempty,datetime=2006-01-02"`
	Hasta string `form:"hasta" binding:"omitempty,datetime=2006-01-02"`
}

// Metricas resume el modelo de negocio en un rango de días.
type Metricas struct {
	Desde           *string              `json:"desde"` // null = desde el comienzo
	Hasta           string               `json:"hasta"`
	Usuarios        MetricasUsuarios     `json:"usuarios"`
	Retencion       MetricasRetencion    `json:"retencion"`
	Colaboracion    MetricasColaboracion `json:"colaboracion"`
	Empresas        MetricasEmpresas     `json:"empresas"`
	ContactosVistos MetricasContactos    `json:"contactos_vistos"`
}

type MetricasUsuarios struct {
	RegistradosPorRol map[string]int `json:"registrados_por_rol"` // registrados en el rango
	Activos7Dias      int            `json:"activos_7_dias"`      // con algún evento en los 7 días que terminan en "hasta"
	Activos30Dias     int            `json:"activos_30_dias"`
}

// MetricasRetencion: de quienes se registraron en el rango, cuántos volvieron justo 1 y 7 días después.
type MetricasRetencion struct {
	Cohorte int       `json:"cohorte"`
	Dia1    Retencion `json:"dia_1"`
	Dia7    Retencion `json:"dia_7"`
}

type Retencion struct {
	Elegibles int     `json:"elegibles"` // de la cohorte, quienes ya cumplieron ese día
	Volvieron int     `json:"volvieron"`
	Tasa      float64 `json:"tasa"` // porcentaje de los elegibles
}

type MetricasColaboracion struct {
	Publicaciones          int `json:"publicaciones"`
	Propuestas             int `json:"propuestas"`
	PropuestasAceptadas    int `json:"propuestas_aceptadas"`    // aceptadas dentro del rango
	ColaboradoresDistintos int `json:"colaboradores_distintos"` // personas que propusieron mejoras en el rango
}

type MetricasEmpresas struct {
	Registradas    int     `json:"registradas"`     // registradas en el rango
	Suscritas      int     `json:"suscritas"`       // de esas, las que han tenido un plan alguna vez
	TasaConversion float64 `json:"tasa_conversion"` // porcentaje de las registradas
	ConPlanActivo  int     `json:"con_plan_activo"` // hoy, sin importar el rango
}

type MetricasContactos struct {
	Total  int `json:"total"`  // eventos: uno por empresa, estudiante y día
	Unicos int `json:"unicos"` // parejas empresa-estudiante distintas
}
