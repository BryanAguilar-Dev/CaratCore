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
