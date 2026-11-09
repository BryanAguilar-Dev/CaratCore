# CaratCore: instrucciones para agentes de IA

El repositorio es la fuente de verdad. Lee este archivo y `docs/decisions.md`
antes de trabajar. No dependas de información que solo exista en un chat.

## Qué es
Herramienta open source (Go) que detecta el hardware, recomienda servicios,
muestra un plan y, tras aprobación del usuario, lo despliega. No esconde la
complejidad: el usuario debe entender qué se hace y por qué.

## Alcance actual: v1 Lite
1. `doctor` (hecho): diagnóstico de solo lectura.
2. Plan de instalación como dato (con motivo por paso y `--dry-run`).
3. Instalar Docker y Docker Compose como pasos del plan (ver D9).
4. Servicios: AdGuard Home y Uptime Kuma (Tailscale solo se detecta).

Todo lo que no esté aquí se apunta en `docs/decisions.md` como idea y no se
implementa hasta cerrar la v1.

## Reglas del proyecto
- No avanzar si el paso anterior no está claro; decisiones justificadas.
- Sin complejidad ni abstracciones innecesarias.
- Un paquete, una responsabilidad. Sin carpetas `utils` ni `helpers`.
- `main.go` sencillo; el núcleo no depende de la CLI.
- Solo librería estándar salvo necesidad justificada y documentada.
- La documentación forma parte del producto.
- No inventar funcionalidades, decisiones ni código sin comprobar.

## Cómo trabajar con Bryan
- Actúa como ingeniero experimentado: cuestiona decisiones si hay mejor
  alternativa y di directamente si algo está mal planteado.
- Respuestas cortas; tareas grandes en pasos validados.
- Bryan está aprendiendo Go, pero prefiere no recibir explicaciones paso a
  paso: el porqué se escribe en `docs/guia.md` (guía de estudio del proyecto,
  actualízala al cerrar cada bloque de trabajo) y en `docs/decisions.md`.
  Antes de cada commit, un resumen breve de qué cambió.
- Pide permiso antes de ejecutar comandos y explica qué hacen.
- Registra cada decisión relevante en `docs/decisions.md`.

## Seguridad
- Nunca pidas ni guardes contraseñas, tokens ni claves.
- Los commits usan el correo `noreply` de GitHub, nunca el personal.
- El instalador se prueba solo en una VM Debian limpia, jamás en Franc (el
  servidor de Bryan, su acceso remoto por Tailscale).
- `doctor` nunca modifica el sistema.

## Comandos
```
go vet ./...
go test ./...
go build -o caratcore ./cmd/caratcore
```
Compilar para Franc desde Windows: `GOOS=linux GOARCH=amd64 go build -o caratcore ./cmd/caratcore`.

## Estado
Ver README.md (qué funciona hoy) y docs/decisions.md (decisiones y pendientes).
