# Registro de decisiones

Estado: **Decidida** (confirmada por Bryan), **Propuesta** (sin confirmar
explícitamente) o **Pendiente**.

## Decididas
- **D1. Lenguaje: Go.**
- **D2. Núcleo independiente de la interfaz.** La CLI es solo una interfaz.
- **D3. Flujo:** system check -> system model -> recommendation -> installation
  plan -> deployment.
- **D4. `doctor` es de solo lectura.** Nunca instala ni modifica nada.
- **D5. Primera plataforma: Debian 13 amd64.**
- **D6. Licencia: Apache-2.0** (2026-11-09).
- **D7. Repositorio nuevo y limpio** (2026-11-09): el historial anterior se
  descartó; el prototipo anterior no se conserva.
- **D8. `doctor` usa solo la librería estándar.** Lee `/proc` y
  `/etc/os-release` en lugar de `gopsutil` o de la salida de `free`/`df`, que
  depende del idioma del sistema. Menos dependencias y más fácil de entender.
- **D9. Orden de la v1** (2026-11-09): primero el plan de instalación como
  dato, después instalar Docker como pasos de ese plan y por último los
  servicios. Así instalar Docker tiene motivo por paso y `--dry-run` desde el
  principio, en vez de ser un instalador aparte que luego habría que rehacer.
- **D10. Solo Debian en la v1; Windows solo para desarrollar** (2026-11-09).
  Windows exigiría Docker Desktop/WSL2 (otro instalador, otra red, otros
  permisos) y no encaja con un servidor siempre encendido que usa el puerto 53.
  Para no cerrar la puerta: el plan es un dato neutro, cada plataforma tiene su
  generador de pasos, y el código dependiente del sistema va en archivos por
  plataforma (`_linux.go` / `_other.go`). En Windows se compila, se pasan los
  tests y se ve el plan con `--dry-run`; la ejecución real solo en Debian.
- **D11. Forma del plan** (2026-11-09), paquete `internal/plan`:
  - Un paso tiene `ID`, `Title`, `Reason`, `Command` y `Root`. `Validate`
    rechaza pasos sin motivo: nada se aprueba sin explicación.
  - `Command` es una lista de argumentos y se ejecuta sin shell. Así no hay
    problemas de comillas ni inyección, y el comando mostrado es el ejecutado.
  - `Root` marca qué necesita privilegios; si se usa `sudo` lo decide quien
    ejecute, no el plan.
  - Un plan vacío es válido: significa "no hay nada que hacer".
  - Construir o mostrar un plan nunca ejecuta nada; ejecutar será otro paquete.
  - Se serializa a JSON (librería estándar) para poder guardarlo o revisarlo.

## Propuestas (por confirmar)
- **P1. Usuario objetivo:** entusiasta principiante y curioso, no un usuario sin
  ninguna base técnica.
- **P2. Despliegue con Docker Compose.** CaratCore genera y explica los
  archivos y ejecuta `docker compose`; no hay motor de despliegue propio.
- **P3. Catálogo de servicios en YAML** incrustado con `go:embed`, y el plan de
  instalación como dato serializable separado de la ejecución.
- **P4. Servicios de la v1:** AdGuard Home (bloqueo de anuncios) y Uptime Kuma
  (monitorización). Tailscale se detecta, no se instala.
- **P5. Ejecutable de terminal**, no un servicio en segundo plano.

## Pendientes
- Estado persistente (archivo o SQLite), rollback y actualizaciones.
- AdGuard Home como DNS de Tailscale: riesgo de perder internet si el servidor
  se apaga; necesita DNS de respaldo.
- Comprobar si el usuario puede usar sudo (hoy solo se detecta que existe).
- Umbrales de RAM y disco de `doctor`: son estimaciones, hay que medirlas con
  los servicios reales.
- Probar los casos de fallo de `doctor` en una máquina real.

## Ideas fuera de la v1
Interfaz web, otras distribuciones, GPU y recomendador avanzado, backups,
monetización (no antes de tener usuarios).
