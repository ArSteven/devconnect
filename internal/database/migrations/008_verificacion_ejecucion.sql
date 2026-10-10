-- 008_verificacion_ejecucion.sql
-- «Pruébalo»: una mejora aceptada queda verificada cuando otra persona ejecuta las dos versiones,
-- la mejora corre sin errores y su salida es distinta de la del original. El código corre en el
-- navegador de quien prueba (o en el Go Playground), nunca en el servidor: aquí solo queda constancia.

ALTER TABLE propuestas_mejora
    ADD COLUMN verificada_en  TIMESTAMPTZ,
    ADD COLUMN verificada_por UUID REFERENCES usuarios(id) ON DELETE SET NULL;
