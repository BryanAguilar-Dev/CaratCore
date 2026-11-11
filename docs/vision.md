# Visión de CaratCore

Hacia dónde podría ir el proyecto más allá de la v1 Lite. Todo lo que hay aquí
son **hipótesis por validar**, no decisiones: las decisiones van en
[decisions.md](decisions.md). Escrito el 2026-11-11, en una conversación
entre Bryan y Claude.

## Qué nos haría distintos

Ya hay muchas herramientas para montar un servidor casero (CasaOS/ZimaOS,
Umbrel, Runtipi, Cosmos, YunoHost; Unraid, de pago). Están citadas de memoria:
hay que estudiarlas antes de decidir nada. Casi todas son gratis, tienen
cientos de apps y comunidades grandes, y funcionan como una **caja negra**:
pulsas "instalar" y algo pasa. Competir en "un clic, más apps" es perder.

CaratCore es una **caja de cristal**:

- **Ves el plan antes de que se ejecute**, con el porqué de cada paso
  (`Why`, `Learn`, `--guided`).
- **Sin atadura**: lo que despliega son `compose.yaml` normales en
  `/opt/caratcore`. Si dejas de usar CaratCore, todo sigue funcionando y lo
  entiendes.
- **Aprendes usándolo**: terminas sabiendo qué es un DNS, un bind mount o
  una clave de firma.

Lema provisional: *el homelab que entiendes y que es tuyo*.

## Para quién

Principiantes curiosos (P1): estudiantes de informática, gente que se
prepara para trabajar en sistemas o redes, y quien tiene un portátil viejo y
no quiere depender de magia. Es un público que paga por aprender.

## El producto final, imaginado

- Núcleo open source: la herramienta y, en la v2, un panel web que muestra y
  aprueba el mismo plan (P10).
- Catálogo corto y cuidado: 15-20 servicios bien explicados, no 300 a
  medias.
- Lo que convierte un juguete en un servidor: actualizaciones, rollback,
  copias de seguridad, HTTPS y monitorización.
- Una ruta de aprendizaje por niveles ("tu primer DNS"...), la idea del modo
  guiado "como en los juegos" llevada a todo el producto.

## Cómo podría dar dinero

1. **Suscripción de servicios, al estilo de Nabu Casa (Home Assistant).** El
   software sigue siendo libre; se cobra por lo que cuesta montarse uno mismo:
   copias de seguridad cifradas fuera de casa y acceso remoto sin
   configuración. Ingreso recurrente, pero exige infraestructura propia y
   confianza con los datos de la gente.
2. **Formación**: cursos o rutas de pago construidos sobre CaratCore. Encaja
   con la caja de cristal y no necesita servidores.
3. **Patrocinios** (GitHub Sponsors): poco, salvo con mucha comunidad.

## Lo que hay que tener presente

- Monetizar herramientas de infraestructura open source es difícil; la
  mayoría de proyectos no lo consigue.
- Sin usuarios no hay nada que monetizar ("monetización, no antes de tener
  usuarios", en las ideas fuera de la v1). El primer paso para validar esto
  no es código: es enseñar la versión publicada donde está el público
  (r/selfhosted, r/homelab) y ver si la caja de cristal le importa a alguien.
- Como portafolio ya funciona: releases probadas, decisiones escritas y
  tests.

## Próximos pasos para validar

1. Estudiar de verdad 3-4 competidores: qué hacen, cómo cobran, qué se
   queja la gente de ellos.
2. Publicar la v0.2 en uno o dos foros con un texto que explique la caja de
   cristal, y apuntar qué preguntan y qué piden.
